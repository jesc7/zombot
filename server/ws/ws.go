package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jesc7/zombot/server/types"
)

type Message struct {
	Payload []byte
}

type WS struct {
	ctx context.Context
	cfg types.Config
	In  chan Message
	Out chan Message
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewWS(ctx context.Context, cfg types.Config) *WS {
	return &WS{
		ctx: ctx,
		cfg: cfg,
		In:  make(chan Message),
		Out: make(chan Message),
	}
}

func (s *WS) Write(pay []byte) error {
	return nil
}

func (s *WS) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		WSHandler(s.ctx, w, r)
	})

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(s.cfg.WS.Port),
		Handler: mux,
	}

	<-s.ctx.Done()

	close(s.In)
	close(s.Out)

	ctxClose, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxClose); err != nil {
		log.Fatalf("Ошибка при выключении: %v", err)
	}
	log.Println("Сервер остановлен")
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
