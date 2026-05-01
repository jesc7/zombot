package ws

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	ctypes "github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	"github.com/jesc7/zombot/server/types"
)

type WebSocketServer struct {
	srv    *http.Server
	jwtKey []byte
	zspy   *websocket.Conn
	b      *bus.Bus
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func NewWebSocketServer(ctx context.Context, cfg types.Config, b *bus.Bus) (*WebSocketServer, error) {
	ch, e := b.Register(types.BUS_WS)
	if e != nil {
		return nil, e
	}

	ws := &WebSocketServer{
		b:      b,
		jwtKey: []byte(cfg.WS.JwtKey),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.handle(ctx, w, r, ch)
	})
	ws.srv = &http.Server{
		Handler: mux,
		Addr:    fmt.Sprintf(":%d", cfg.WS.Port),
	}
	return ws, nil
}

func (ws *WebSocketServer) Run(ctx context.Context) error {
	go func() {
		if e := ws.srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Fatalf("WebSocket server error: %v", e)
		}
	}()

	log.Println("WebSocket server started, here the tokens:")
	for k, v := range map[clientType]string{ct_ZSPY: "zspy"} {
		jwt, e := jwtGenerate(ws.jwtKey, k)
		log.Printf("%s=%s (%v)\n", v, jwt, e)
	}

	<-ctx.Done()

	ctxClose, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if e := ws.srv.Shutdown(ctxClose); e != nil {
		return e
	}
	return ctx.Err()
}

func (ws *WebSocketServer) handle(ctx context.Context, w http.ResponseWriter, r *http.Request, ch chan shared.Envelope) {
	auth := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(auth, "Bearer ")
	if auth == "" || tokenStr == auth {
		http.Error(w, "Auth header expected", http.StatusUnauthorized)
		return
	}

	claims := &Claims{}
	token, e := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) { return ws.jwtKey, nil })
	if e != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	conn, e := upgrader.Upgrade(w, r, nil)
	if e != nil {
		http.Error(w, "Upgrade: WebSocket", http.StatusUpgradeRequired)
		return
	}

	log.Printf("Connect %s", conn.RemoteAddr())
	defer log.Printf("Disconnect %s", conn.RemoteAddr())

	switch claims.Type {
	case ct_ZSPY:
		if ws.zspy != nil {
			http.Error(w, "ZSpy already connected", http.StatusNotAcceptable)
			return
		}
		ws.handleSpy(ctx, conn, ch)

	default:
		return
	}
}

type clientType string

type Claims struct {
	Type clientType `json:"type"`
	jwt.RegisteredClaims
}

func jwtGenerate(key []byte, ct clientType) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256,
		&Claims{
			Type: ct,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(ctypes.ClearTime(time.Now()).Add(time.Hour * 24 * 365 * 10)), //10 years
			},
		},
	).SignedString(key)
}
