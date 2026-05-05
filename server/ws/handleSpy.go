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

var BUS_NAMES = []string{types.BUS_BOTMAX, types.BUS_BOTTG}

func (ws *WebSocketServer) handleSpy(ctx context.Context, conn *websocket.Conn, ch chan any) {
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
			for _, v := range BUS_NAMES {
				ws.b.Write(v, env)
			}
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

		case msg := <-ch: //требуется переслать сообщение клиенту zspy
			switch mt := msg.(type) {
			case shared.Envelope:
				if e := shared.Write(ws.zspy, mt); e != nil {
					log.Println(e)
				}
			}
		}
	}
}
