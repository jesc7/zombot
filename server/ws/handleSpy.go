package ws

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/server/types"
)

const ct_ZSPY clientType = "zspy"

func (ws *WebSocketServer) handleSpy(ctx context.Context, conn *websocket.Conn, ch chan shared.Envelope) {

	log.Printf("Connect: %s / %s", conn.LocalAddr(), conn.RemoteAddr())

	ws.zspy = conn
	defer func() {
		ws.zspy.Close()
		ws.zspy = nil
	}()
	readError := make(chan struct{})

	go func() {
		defer close(readError)

		for {
			env, e := shared.Read(ws.zspy)
			if e != nil { //ошибка чтения сокета - выходим
				return
			}
			ws.b.Write(types.BUS_BOT, env)
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

		case env := <-ch: //наконец-то делаем что-то полезное
			if e := shared.Write(ws.zspy, env); e != nil {
				log.Println(e)
			}
		}
	}
}
