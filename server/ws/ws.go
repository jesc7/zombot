package ws

import (
	"context"
	"net/http"

	"github.com/jesc7/zombot/server/types"
)

type Message struct {
	Payload []byte
}

type WS struct {
	In  chan Message
	Out chan Message
}

func NewWS(ctx context.Context, cfg types.Config) *WS {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		WSHandler(ctx, w, r)
	})
	return &WS{}
}

func (s *WS) Run(ctx context.Context) {
}
