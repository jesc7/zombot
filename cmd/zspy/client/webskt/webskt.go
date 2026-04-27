package webskt

import (
	"context"
	"encoding/json"
	"errors"
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
	ch     chan shared.MessageText
	conn   *websocket.Conn
}

func NewWebSocketClient(cfg types.Config) *WebSocketClient {
	return &WebSocketClient{
		host:   url.URL{Scheme: "ws", Host: cfg.Addr, Path: "/ws"},
		header: http.Header{"Authorization": []string{"Bearer " + cfg.Token}},
		ch:     make(chan shared.MessageText),
	}
}

func (ws *WebSocketClient) WriteText(text string) {
	defer recover()
	ws.ch <- shared.MessageText{
		Text: text,
	}
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
			case shared.MT_DUTY:
				var d shared.MessageDuties
				if e = json.Unmarshal(raw, &d); e != nil {
					continue
				}
				//d.A = duties.Duty(db, nil, d.Q)
				write(conn, d)
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

func write(conn *websocket.Conn, v any) error {
	raw, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return conn.WriteMessage(websocket.TextMessage, raw)
}

func read(conn *websocket.Conn) (m shared.Message, raw []byte, e error) {
	mt, raw, e := conn.ReadMessage()
	if e != nil {
		return
	}
	switch mt {
	case websocket.TextMessage:
		e = json.Unmarshal(raw, &m)
		return m, raw, e

	case websocket.PingMessage, websocket.PongMessage:
		m.Type = shared.MT_UNDEFINED
		return

	default:
		return m, raw, errors.New("Undefined message")
	}
}
