package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
)

type Config struct {
	Addr  string
	Token string
}

type MessageType int

const (
	MT_UNDEFINED MessageType = iota - 1
	MT_PING
)

type Message struct {
	Type MessageType
}

func Start(ctx context.Context, service bool) error {
	bin, e := runPath(service)
	if e != nil {
		return e
	}

	f, e := os.ReadFile(filepath.Join(filepath.Dir(bin), "cfg.json"))
	if e != nil {
		return e
	}
	var cfg Config
	if e = json.Unmarshal(f, &cfg); e != nil {
		return e
	}

	u := url.URL{Scheme: "ws", Host: cfg.Addr, Path: "/ws"}
	header := http.Header{"Authorization": []string{"Bearer " + cfg.Token}}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			conn, _, e := websocket.DefaultDialer.DialContext(ctx, u.String(), header)
			if e != nil {
				select {
				case <-ctx.Done():
					return ctx.Err()

				case <-time.After(5 * time.Second):
					continue
				}
			}
			handleConnection(ctx, conn)
		}
	}
}

func handleConnection(ctx context.Context, conn *websocket.Conn) {
	defer conn.Close()
	done := make(chan struct{})

	_read := func(conn *websocket.Conn) (m Message, raw []byte, e error) {
		mt, raw, e := conn.ReadMessage()
		if e != nil {
			return
		}
		switch mt {
		case websocket.TextMessage:
			e = json.Unmarshal(raw, &m)
			return m, raw, e

		case websocket.PongMessage:
			m.Type = MT_UNDEFINED
			return

		default:
			return m, raw, errors.New("Undefined message")
		}
	}

	go func() {
		defer close(done)

		for {
			msg, raw, e := _read(conn)
			if e != nil {
				return
			}

			switch msg.Type {
			case MT_PING:
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

		case <-tPing.C: //ping соединения
			if e := conn.WriteMessage(websocket.PingMessage, nil); e != nil {
				return
			}
		}
	}
}
