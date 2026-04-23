package ws

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jesc7/zombot/server/types"
)

type Message struct {
	Payload []byte
}

type WS struct {
	cfg types.Config
	In  chan Message
	Out chan Message
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewWS(ctx context.Context, cfg types.Config) *WS {
	return &WS{
		cfg: cfg,
		In:  make(chan Message),
		Out: make(chan Message),
	}
}

func (s *WS) Run(ctx context.Context) {
	mux := http.NewServeMux()
	// Оборачиваем обработчик, чтобы передать контекст
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		WSHandler(ctx, w, r)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		WSHandler(ctx, w, r)
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
