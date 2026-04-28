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
	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	"github.com/jesc7/zombot/server/types"
)

type WebSocketServer struct {
	srv    *http.Server
	jwtKey []byte
	spy    *websocket.Conn
	b      *bus.Bus
	ChOut  chan shared.Envelope
	chIn   <-chan shared.Envelope
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func NewWebSocketServer(ctx context.Context, cfg types.Config) *WebSocketServer {
	ws := &WebSocketServer{
		jwtKey: []byte(cfg.WS.JwtKey),
		ChOut:  make(chan shared.Envelope),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.handle(ctx, w, r)
	})
	ws.srv = &http.Server{
		Handler: mux,
		Addr:    fmt.Sprintf(":%d", cfg.WS.Port),
	}
	return ws
}

func (ws *WebSocketServer) Run(ctx context.Context, ch <-chan shared.Envelope) error {
	defer close(ws.ChOut)
	ws.chIn = ch

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

func (ws *WebSocketServer) Write(env shared.Envelope) {
	defer recover()
	ws.ChOut <- env
}

func (ws *WebSocketServer) handle(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

	switch claims.Type {
	case ct_ZSPY:
		ws.handleSpy(ctx, w, r)

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
				ExpiresAt: jwt.NewNumericDate(time.Now().Truncate(24 * time.Hour).Add(time.Hour * 24 * 365 * 10)), //10 years
			},
		},
	).SignedString(key)
}
