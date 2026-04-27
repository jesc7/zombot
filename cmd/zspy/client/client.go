package client

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jesc7/zombot/cmd/zspy/client/jp/duties"
	"github.com/jesc7/zombot/cmd/zspy/client/webapi"
	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type Config struct {
	Addr  string
	Token string
}

func Start(ctx context.Context, service bool) error {
	cwd, e := runPath(service)
	if e != nil {
		return e
	}

	f, e := os.ReadFile(filepath.Join(filepath.Dir(cwd), "cfg.json"))
	if e != nil {
		return e
	}
	var cfg Config
	if e = json.Unmarshal(f, &cfg); e != nil {
		return e
	}

	wg := &sync.WaitGroup{}
	wg.Go(func() { //run WebSocket server
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
	})

	wa := webapi.NewServer()
	wg.Go(func() { //run Max bot
		defer func() {
			log.Println("Max bot has been stopped")
		}()
		wa.Run(ctx)
	})

	wg.Wait()
	return nil
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
				d.A = duties.Duty(db, nil, d.Q)
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
