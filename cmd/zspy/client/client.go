package client

import (
	"context"
	"encoding/json"
	"log"
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

	return nil
}

func handleConnection(ctx context.Context, conn *websocket.Conn) {
	defer conn.Close()
	done := make(chan struct{})

	// Горутина чтения
	go func() {
		defer close(done)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Ошибка чтения: %v", err)
				return
			}
			log.Printf("Сообщение: %s", msg)
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Изящное закрытие (Close Handshake)
			log.Println("Закрытие соединения по контексту...")
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Ошибка при закрытии:", err)
			}
			return
		case <-done:
			log.Println("Соединение разорвано сервером")
			return
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				log.Printf("Ошибка записи: %v", err)
				return
			}
		}
	}
}
