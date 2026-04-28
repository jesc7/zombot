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
	"github.com/jesc7/zombot/server/types"
)

/*type Message struct {
	Payload []byte
}*/

type WebSocketServer struct {
	srv    *http.Server
	jwtKey []byte
	spy    *websocket.Conn
}

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

func NewWebSocketServer(cfg types.Config) *WebSocketServer {
	ws := &WebSocketServer{
		jwtKey: []byte(cfg.WS.JwtKey),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handle(ws, w, r)
	})
	ws.srv = &http.Server{
		Handler: mux,
		Addr:    fmt.Sprintf(":%d", cfg.WS.Port),
	}
	return ws
}

func (ws *WebSocketServer) Read() ([]byte, error) {
	return nil, nil
}

func (ws *WebSocketServer) Write(pay []byte) error {
	return nil
}

type ClientType string

const (
	CT_ZSPY ClientType = "zspy"
)

type Claims struct {
	Type ClientType `json:"type"`
	jwt.RegisteredClaims
}

func jwtGenerate(key []byte, ct ClientType) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256,
		&Claims{
			Type: ct,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Truncate(24 * time.Hour).Add(time.Hour * 24 * 365 * 10)), //10 years
			},
		},
	).SignedString(key)
}

func handle(ws *WebSocketServer, w http.ResponseWriter, r *http.Request) {
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
	case CT_ZSPY:
		if ws.spy != nil {
			http.Error(w, "ZSpy already connected", http.StatusNotAcceptable)
			return
		}

	default:
		return
	}

	if ws.spy, e = upgrader.Upgrade(w, r, nil); e != nil {
		http.Error(w, "Upgrade: WebSocket", http.StatusUpgradeRequired)
		return
	}
	defer func() {
		ws.spy.Close()
		ws.spy = nil
	}()

	// Канал для передачи сообщений от "бизнес-логики" к клиенту
	outbound := make(chan string)
	// Канал для отслеживания ошибок чтения
	readError := make(chan struct{})

	// 1. Горутина для ЧТЕНИЯ (Incoming TextMessages)
	go func() {
		for {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Read error: %v", err)
				close(readError)
				return
			}

			// Обрабатываем только текстовые сообщения
			if messageType == websocket.TextMessage {
				log.Printf("Входящее: %s", string(payload))

				// Пример логики: отвечаем клиенту через канал
				outbound <- fmt.Sprintf("Сервер получил: %s", string(payload))
			}
		}
	}()

	// 2. Основной цикл записи и мониторинга контекста
	for {
		select {
		case msg := <-outbound:
			// ОТПРАВКА (Outgoing TextMessage)
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				log.Printf("Write error: %v", err)
				return
			}

		case <-readError:
			log.Println("Завершаем работу: ошибка чтения или клиент ушел")
			return

			/*case <-ctx.Done():
			log.Println("Внешний контекст отменен: закрываем соединение")
			// Вежливо прощаемся с клиентом
			conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server Shutdown"),
				time.Now().Add(time.Second),
			)
			return*/
		}
	}
}

func (ws *WebSocketServer) Run(ctx context.Context) {
	go func() {
		if e := ws.srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Fatalf("WebSocket server error: %v", e)
		}
	}()

	log.Println("WebSocket server started, here the tokens:")
	for k, v := range map[ClientType]string{CT_ZSPY: "zspy"} {
		jwt, e := jwtGenerate(ws.jwtKey, k)
		log.Printf("%s=%s (%v)\n", v, jwt, e)
	}

	<-ctx.Done()

	ctxClose, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if e := ws.srv.Shutdown(ctxClose); e != nil {
		log.Fatalf("WebSocket server shutdown error: %v", e)
	}
}
