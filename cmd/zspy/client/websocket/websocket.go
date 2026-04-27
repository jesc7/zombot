package websocket

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketClient struct{}

func NewClient() *WebSocketClient {
	return &WebSocketClient{}
}

func (ws *WebSocketClient) Run(ctx context.Context) {
	u := url.URL{Scheme: "ws", Host: cfg.Addr, Path: "/ws"}
	header := http.Header{"Authorization": []string{"Bearer " + cfg.Token}}

	for {
		select {
		case <-ctx.Done():
			return

		default:
			conn, _, e := websocket.DefaultDialer.DialContext(ctx, u.String(), header)
			if e != nil {
				select {
				case <-ctx.Done():
					return

				case <-time.After(5 * time.Second):
					continue
				}
			}
			handle(ctx, conn)
		}
	}
}
