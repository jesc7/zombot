package webskt

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
	ch     chan any
	conn   *websocket.Conn
}

func NewWebSocketClient(cfg types.Config) *WebSocketClient {
	return &WebSocketClient{
		host:   url.URL{Scheme: "ws", Host: cfg.Addr, Path: "/ws"},
		header: http.Header{"Authorization": []string{"Bearer " + cfg.Token}},
		ch:     make(chan any),
	}
}

func (ws *WebSocketClient) Write(msg any) {
	defer recover()
	ws.ch <- msg
}

func (ws *WebSocketClient) Run(ctx context.Context) {
	defer close(ws.ch)

	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-ws.ch:
			if ws.conn != nil {
				write(ws.conn, msg)
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
			ws.handle(ctx)
		}
	}
}

func (ws *WebSocketClient) handle(ctx context.Context) {
	defer ws.conn.Close()
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			raw, e := read(ws.conn)
			if e != nil {
				return
			}

			switch m := msg.(type) {
			case shared.MessageDuties:
				var d shared.MessageDuties
				if e = json.Unmarshal(raw, &d); e != nil {
					continue
				}
				//d.A = duties.Duty(db, nil, d.Q)
				write(ws.conn, d)
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

func write(conn *websocket.Conn, env shared.Envelope) error {
	data, e := json.Marshal(env)
	if e != nil {
		return e
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

func read(conn *websocket.Conn) (shared.Envelope, error) {
	mt, data, e := conn.ReadMessage()
	if e != nil {
		return shared.Envelope{}, e
	}

	switch mt {
	case websocket.TextMessage:
		var env shared.Envelope
		e = json.Unmarshal(data, &env)
		return env, e

	default:
		return shared.Envelope{}, nil
	}
}
