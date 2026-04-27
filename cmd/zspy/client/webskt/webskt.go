package webskt

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jesc7/zombot/cmd/zspy/client/jp/duties"
	"github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type WebSocketClient struct {
	host   url.URL
	header http.Header
	ch     chan shared.Envelope
	conn   *websocket.Conn
}

func NewWebSocketClient(cfg types.Config) *WebSocketClient {
	return &WebSocketClient{
		host:   url.URL{Scheme: "ws", Host: cfg.Addr, Path: "/ws"},
		header: http.Header{"Authorization": []string{"Bearer " + cfg.Token}},
		ch:     make(chan shared.Envelope),
	}
}

func (ws *WebSocketClient) Write(env shared.Envelope) {
	defer recover()
	ws.ch <- env
}

func (ws *WebSocketClient) Run(ctx context.Context, db *sql.DB) {
	defer close(ws.ch)

	for {
		select {
		case <-ctx.Done():
			return

		case env := <-ws.ch:
			if ws.conn != nil {
				write(ws.conn, env)
			}

		default:
			var e error
			ws.conn, _, e = websocket.DefaultDialer.DialContext(ctx, ws.host.String(), ws.header)
			if e != nil {
				select {
				case <-ctx.Done():
					return

				case <-time.After(5 * time.Second):
					continue
				}
			}
			ws.handle(ctx, db)
		}
	}
}

func (ws *WebSocketClient) handle(ctx context.Context, db *sql.DB) {
	defer ws.conn.Close()
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			env, e := read(ws.conn)
			if e != nil {
				return
			}

			switch env.Type {
			case "MessageDuties":
				d, e := shared.Unpack[shared.MessageDuties](env)
				if e != nil {
					continue
				}
				d.A, e = duties.Duty(ctx, db, d.Q)
				if e != nil {
					continue
				}
				env, e = shared.Pack(env.Type, d)
				if e != nil {
					continue
				}
				write(ws.conn, env)
			}
		}
	}()

	tPing := time.NewTicker(10 * time.Second)
	defer tPing.Stop()

	for {
		select {
		case <-ctx.Done(): //выход по контексту
			ws.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return

		case <-done: //сервер закрыл соединение
			return

		case <-tPing.C: //пингуем соединение
			if e := ws.conn.WriteMessage(websocket.PingMessage, nil); e != nil {
				return
			}
		}
	}
}
