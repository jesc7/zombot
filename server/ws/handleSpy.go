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

func (ws *WebSocketServer) handleSpy(ctx context.Context, conn *websocket.Conn) {
	if ws.zspy != nil {
		http.Error(w, "ZSpy already connected", http.StatusNotAcceptable)
		return
	}

	var e error
	if ws.zspy, e = upgrader.Upgrade(w, r, nil); e != nil {
		http.Error(w, "Upgrade: WebSocket", http.StatusUpgradeRequired)
		return
	}

	defer func() {
		ws.zspy.Close()
		ws.zspy = nil
	}()
	readError := make(chan struct{})

	go func() {
		defer close(readError)

		for {
			env, e := shared.Read(ws.zspy)
			if e != nil {
				return
			}
			ws.b.Write("bot", env)
		}
	}()

	for {
		select {
		case <-ctx.Done(): //контекст отменен - выходим
			ws.zspy.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server Shutdown"),
				time.Now().Add(time.Second),
			)
			return

		case <-readError: //ошибка чтения сокета - выходим
			return

		case env := <-ws.chIn: //наконец-то делаем что-то полезное
			if e := shared.Write(ws.zspy, env); e != nil {
				log.Println(e)
			}
		}
	}
}
