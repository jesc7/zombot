package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/client/webskt"
	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type WebServer struct {
	srv *http.Server
	skt *webskt.WebSocketClient
}

func NewWebServer(skt *webskt.WebSocketClient) *WebServer {
	mux := http.NewServeMux()
	//скрипт asterisk 192.168.67.11/etc/asterisk/IgorBot.php шлет запрос вида 'ip:8089/call?phone=XXXXXX'
	mux.HandleFunc("/call", func(w http.ResponseWriter, r *http.Request) {
		v, ok := r.URL.Query()["phone"]
		if !ok {
			return
		}
		skt.WriteText(fmt.Sprintf("📞 Вам звонили%s: <b>%s</b>\n", types.Iif(strings.HasPrefix(v[0], "8800 "), " на 8800", ""), v[0]))
	})

	//сообщения от ZSrv
	mux.HandleFunc("/zsrv", func(w http.ResponseWriter, r *http.Request) {
		var msg shared.MessageZSRV
		if e := json.NewDecoder(r.Body).Decode(&msg); e != nil {
			http.Error(w, e.Error(), http.StatusBadRequest)
			return
		}
		if strings.Count(msg.Text, "\n") != 0 {
			msg.Text = "\n" + msg.Text
		}
		switch msg.Status {
		case shared.ZMSG_WARN:
			msg.Text = fmt.Sprintf("⚠️ <i>zsrv %s беспокоится</i>\n%s", msg.Caption, msg.Text)
		case shared.ZMSG_PANIC:
			msg.Text = fmt.Sprintf("🆘 <i>zsrv %s паникует</i>\n%s", msg.Caption, msg.Text)
		default:
			msg.Text = fmt.Sprintf("ℹ <i>zsrv %s информирует</i>\n%s", msg.Caption, msg.Text)
		}
		skt.WriteText(msg.Text)
		w.WriteHeader(http.StatusOK)
	})

	return &WebServer{
		srv: &http.Server{
			Handler: mux,
			Addr:    ":8089",
		},
	}
}

func (ws *WebServer) Run(ctx context.Context) {
	go func() {
		if e := ws.srv.ListenAndServe(); e != http.ErrServerClosed {
			log.Println("Http server error:", e)
		}
	}()
	<-ctx.Done()

	ctxClose, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if e := ws.srv.Shutdown(ctxClose); e != nil {
		log.Println("Http server shutdown error:", e)
	}
}
