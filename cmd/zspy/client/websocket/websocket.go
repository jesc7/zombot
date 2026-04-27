package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type WebSocketClient struct {
	host   url.URL
	header http.Header
}

func NewWebSocketClient(cfg types.Config) *WebSocketClient {
	return &WebSocketClient{
		host:   url.URL{Scheme: "ws", Host: cfg.Addr, Path: "/ws"},
		header: http.Header{"Authorization": []string{"Bearer " + cfg.Token}},
	}
}

func (ws *WebSocketClient) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		default:
			conn, _, e := websocket.DefaultDialer.DialContext(ctx, ws.host.String(), ws.header)
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

func handle(ctx context.Context, conn *websocket.Conn) {
	defer conn.Close()
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			msg, raw, e := read(conn)
			if e != nil {
				return
			}

			switch msg.Type {
			case shared.MT_UNDEFINED:

			case shared.MT_DUTY:
				var d shared.MessageDuties
				if e = json.Unmarshal(raw, &d); e != nil {
					continue
				}
				//d.A = duties.Duty(db, nil, d.Q)
				write(conn, d)

			default:
				_ = raw
			}
		}
	}()

	tPing := time.NewTicker(10 * time.Second)
	defer tPing.Stop()

	for {
		select {
		case <-ctx.Done(): //выход по контексту
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return

		case <-done: //сервер закрыл соединение
			return

		case <-tPing.C: //пингуем соединение
			if e := conn.WriteMessage(websocket.PingMessage, nil); e != nil {
				return
			}
		}
	}
}
