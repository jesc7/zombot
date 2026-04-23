package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jesc7/zombot/server/types"
)

type Message struct {
	Payload []byte
}

type WS struct {
	ctx context.Context
	cfg types.Config
	In  chan Message
	Out chan Message
}

func NewWS(ctx context.Context, cfg types.Config) *WS {
	return &WS{
		ctx: ctx,
		cfg: cfg,
		In:  make(chan Message),
		Out: make(chan Message),
	}
}

func (s *WS) Write(pay []byte) error {
	return nil
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Канал для передачи сообщений от "бизнес-логики" к клиенту
	outbound := make(chan string)
	// Канал для отслеживания ошибок чтения
	readError := make(chan struct{})

	// 1. Горутина для ЧТЕНИЯ (Incoming TextMessages)
	go func() {
		for {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Read error: %v", err)
				close(readError)
				return
			}

			// Обрабатываем только текстовые сообщения
			if messageType == websocket.TextMessage {
				log.Printf("Входящее: %s", string(payload))

				// Пример логики: отвечаем клиенту через канал
				outbound <- fmt.Sprintf("Сервер получил: %s", string(payload))
			}
		}
	}()

	// 2. Основной цикл записи и мониторинга контекста
	for {
		select {
		case msg := <-outbound:
			// ОТПРАВКА (Outgoing TextMessage)
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				log.Printf("Write error: %v", err)
				return
			}

		case <-readError:
			log.Println("Завершаем работу: ошибка чтения или клиент ушел")
			return

		case <-ctx.Done():
			log.Println("Внешний контекст отменен: закрываем соединение")
			// Вежливо прощаемся с клиентом
			conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server Shutdown"),
				time.Now().Add(time.Second),
			)
			return
		}
	}
}

func (s *WS) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handler(s, w, r)
	})

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(s.cfg.WS.Port),
		Handler: mux,
	}

	<-s.ctx.Done()

	close(s.In)
	close(s.Out)

	ctxClose, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxClose); err != nil {
		log.Fatalf("Ошибка при выключении: %v", err)
	}
	log.Println("Сервер остановлен")
}

func write(conn *websocket.Conn, v any) error {
	raw, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return conn.WriteMessage(websocket.TextMessage, raw)
}

func read(conn *websocket.Conn) (m Message, raw []byte, e error) {
	mt, raw, e := conn.ReadMessage()
	if e != nil {
		return
	}
	switch mt {
	case websocket.TextMessage:
		e = json.Unmarshal(raw, &m)
		return m, raw, e

	case websocket.PingMessage, websocket.PongMessage:
		return

	default:
		return m, raw, errors.New("Undefined message")
	}
}
