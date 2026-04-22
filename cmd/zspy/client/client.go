package client

import (
	"bytes"
	"context"
	"encoding/json"
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
	MT_PING MessageType = iota
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

	_read := func(conn *websocket.Conn) (m Message, raw string, e error) {
		_, buf, e := conn.ReadMessage()
		if e != nil {
			return
		}
		e = json.NewDecoder(bytes.NewReader(buf)).Decode(&m)
		return m, string(buf), e
	}

	go func() {
		defer close(done)

		for {
			msg, raw, e := _read(conn) //conn.ReadJSON(&msg)
			if e != nil {
				return
			}
			_, _ = msg, raw
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

		case <-tPing.C: //ошибка отправки сообщения ping
			if e := conn.WriteMessage(websocket.TextMessage, []byte("ping")); e != nil {
				return
			}
		}
	}
}
