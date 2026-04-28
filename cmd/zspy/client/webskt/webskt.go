package webskt

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jesc7/zombot/cmd/zspy/client/jp/duties"
	"github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
	_ "github.com/nakagami/firebirdsql"
)

type WebSocketClient struct {
	host   url.URL
	header http.Header
	ch     chan shared.Envelope
	conn   *websocket.Conn
}

func NewWebSocketClient(cfg types.Config) *WebSocketClient {
	return &WebSocketClient{
		host:   url.URL{Scheme: "ws", Host: cfg.Host, Path: "/ws"},
		header: http.Header{"Authorization": []string{"Bearer " + cfg.Token}},
		ch:     make(chan shared.Envelope),
	}
}

func (ws *WebSocketClient) Write(env shared.Envelope) {
	defer recover()
	ws.ch <- env
}

func (ws *WebSocketClient) Run(ctx context.Context, cfg types.Config) error {
	defer close(ws.ch)

	db, e := sql.Open(cfg.DB.Driver, cfg.DB.ConnStr)
	if e != nil {
		return e
	}
	defer db.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case env := <-ws.ch:
			if ws.conn != nil {
				shared.Write(ws.conn, env)
			}

		default:
			var e error
			ws.conn, _, e = websocket.DefaultDialer.DialContext(ctx, ws.host.String(), ws.header)
			if e != nil {
				log.Println(e)
				select {
				case <-ctx.Done():
					return ctx.Err()

				case <-time.After(10 * time.Second):
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
			env, e := shared.Read(ws.conn)
			if e != nil {
				return
			}

			switch env.Type {
			case shared.MT_MessageDuties:
				dut, e := shared.Unpack[shared.MessageDuties](env)
				if e != nil {
					continue
				}
				dut.A, e = duties.Duty(ctx, db, dut.Q)
				if e != nil {
					continue
				}
				env, e = shared.Pack(env.Type, dut)
				if e != nil {
					continue
				}
				shared.Write(ws.conn, env)
			}
		}
	}()

	tPing := time.NewTicker(10 * time.Second)
	defer tPing.Stop()

	for {
		select {
		case <-ctx.Done():
			ws.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return

		case <-done:
			return

		case <-tPing.C:
			if e := ws.conn.WriteMessage(websocket.PingMessage, nil); e != nil {
				return
			}
		}
	}
}
