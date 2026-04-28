package ws

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jesc7/zombot/cmd/zspy/shared"
)

const ct_ZSPY clientType = "zspy"

func (ws *WebSocketServer) handleSpy(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if ws.spy != nil {
		http.Error(w, "ZSpy already connected", http.StatusNotAcceptable)
		return
	}

	var e error
	if ws.spy, e = upgrader.Upgrade(w, r, nil); e != nil {
		http.Error(w, "Upgrade: WebSocket", http.StatusUpgradeRequired)
		return
	}

	defer func() {
		ws.spy.Close()
		ws.spy = nil
	}()
	readError := make(chan struct{})

	go func() {
		defer close(readError)

		for {
			env, e := shared.Read(ws.spy)
			if e != nil {
				return
			}

			switch env.Type {
			case shared.MT_MessageText:
			}
		}
	}()

	for {
		select {
		case <-ctx.Done(): //контекст отменен - выходим
			ws.spy.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server Shutdown"),
				time.Now().Add(time.Second),
			)
			return

		case <-readError: //ошибка чтения сокета - выходим
			return

		case env := <-ws.ch: //наконец-то делаем что-то полезное
			if e := shared.Write(ws.spy, env); e != nil {
				log.Println(e)
			}
		}
	}
}
