package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Payload []byte
}

type WS struct {
	cfg types.Config
	in  chan Message
	out chan Message
}

func NewWS(cfg types.Config) *WS {
	return &WS{
		cfg: cfg,
		in:  make(chan Message),
		out: make(chan Message),
	}
}

func (s *WS) Read() ([]byte, error) {
	return nil, nil
}

func (s *WS) Write(pay []byte) error {
	return nil
}

type connType int
type connInfo struct {
	Type connType
}

const (
	CT_ZSPY connType = iota
)

var typesCnt = map[connType]uint8{
	CT_ZSPY: 1,
}

var (
	types    = map[connType]uint8{}
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conns    = map[*websocket.Conn]connInfo{}
)

func connCheck()

func handler(w http.ResponseWriter, r *http.Request) {
	conn, e := upgrader.Upgrade(w, r, nil)
	if e != nil {
		log.Printf("Upgrade error: %v", e)
		w.WriteHeader(http.StatusUpgradeRequired)
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

func (s *WS) Run(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handler(s, w, r)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.WS.Port),
		Handler: mux,
	}

	go func() {
		if e := server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Fatalf("WebSocket server error: %v", e)
		}
	}()

	<-ctx.Done()

	close(s.in)
	close(s.out)

	ctxClose, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if e := server.Shutdown(ctxClose); e != nil {
		log.Fatalf("WebSocket server shutdown error: %v", e)
	}
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
