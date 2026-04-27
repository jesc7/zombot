package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/jesc7/zombot/server/types"
)

type Message struct {
	Payload []byte
}

type WS struct {
	cfg     types.Config
	connSpy *websocket.Conn
}

var (
	jwtKey   []byte
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

func NewWS(cfg types.Config) *WS {
	jwtKey = []byte(cfg.WS.JwtKey)
	return &WS{
		cfg: cfg,
	}
}

func (s *WS) Read() ([]byte, error) {
	return nil, nil
}

func (s *WS) Write(pay []byte) error {
	return nil
}

type ClientType string

const (
	CT_ZSPY ClientType = "zspy"
)

type Claims struct {
	Type ClientType `json:"client_type"`
	jwt.RegisteredClaims
}

func jwtGenerate(ct ClientType) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256,
		&Claims{
			Type: ct,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 365 * 10)), //10 years
			},
		},
	).SignedString(jwtKey)
}

func handle(ws *WS, w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(auth, "Bearer ")
	if auth == "" || tokenStr == auth {
		http.Error(w, "Auth header expected", http.StatusUnauthorized)
		return
	}

	claims := &Claims{}
	token, e := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) { return jwtKey, nil })
	if e != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	switch claims.Type {
	case CT_ZSPY:
		if ws.connSpy != nil {
			http.Error(w, "ZSpy already connected", http.StatusNotAcceptable)
			return
		}

	default:
		return
	}

	conn, e := upgrader.Upgrade(w, r, nil)
	if e != nil {
		http.Error(w, "Upgrade: WebSocket", http.StatusUpgradeRequired)
		return
	}
	defer conn.Close()

	ws.connSpy = conn
	defer func() {
		ws.connSpy = nil
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

		case <-ctx.Done():
			log.Println("Внешний контекст отменен: закрываем соединение")
			// Вежливо прощаемся с клиентом
			conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server Shutdown"),
				time.Now().Add(time.Second),
			)
			return
		}
	}
}

func (s *WS) Run(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handle(s, w, r)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.WS.Port),
		Handler: mux,
	}

	go func() {
		if e := server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Fatalf("WebSocket server error: %v", e)
		}
	}()
	log.Println("WebSocket server started, here tokens:")
	jwtZSpy, e := jwtGenerate(CT_ZSPY)
	log.Printf("zspy=%s (%v)\n", jwtZSpy, e)

	<-ctx.Done()

	ctxClose, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if e := server.Shutdown(ctxClose); e != nil {
		log.Fatalf("WebSocket server shutdown error: %v", e)
	}
}

func write(conn *websocket.Conn, v any) error {
	raw, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return conn.WriteMessage(websocket.TextMessage, raw)
}

func read(conn *websocket.Conn) (m Message, raw []byte, e error) {
	mt, raw, e := conn.ReadMessage()
	if e != nil {
		return
	}
	switch mt {
	case websocket.TextMessage:
		e = json.Unmarshal(raw, &m)
		return m, raw, e

	case websocket.PingMessage, websocket.PongMessage:
		return

	default:
		return m, raw, errors.New("Undefined message")
	}
}
