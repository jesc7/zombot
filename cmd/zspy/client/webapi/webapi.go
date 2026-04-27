package webapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

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
		skt.Write(shared.MessageCall{Phone: v[0]})
	})

	//сообщения от ZSrv
	mux.HandleFunc("/zsrv", func(w http.ResponseWriter, r *http.Request) {
		var msg shared.MessageZSRV
		if e := json.NewDecoder(r.Body).Decode(&msg); e != nil {
			http.Error(w, e.Error(), http.StatusBadRequest)
			return
		}
		skt.Write(msg)
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
