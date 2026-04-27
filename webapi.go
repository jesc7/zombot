package main

/*
import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	maxbot "github.com/jesc7/zombot/server/max/bot"
	"github.com/jesc7/zombot/server/types"
)

type WebServer struct {
	srv *http.Server
}

func NewServer(ctx context.Context, cfg types.Config, bot *maxbot.Bot) *WebServer {
	mux := &http.ServeMux{}
	//скрипт asterisk 192.168.67.11/etc/asterisk/IgorBot.php шлет запрос вида 'ip:8089/call?phone=XXXXXX'
	mux.HandleFunc("/call", func(w http.ResponseWriter, r *http.Request) {
		v, ok := r.URL.Query()["phone"]
		if !ok {
			return
		}
		bot.SendCall(v[0])
	})

	//сообщения от ZSrv
	mux.HandleFunc("/zsrv", func(w http.ResponseWriter, r *http.Request) {
		var msg types.ZSrvMessage
		if e := json.NewDecoder(r.Body).Decode(&msg); e != nil {
			http.Error(w, e.Error(), http.StatusBadRequest)
			return
		}
		bot.SendZSrv(msg)
		w.WriteHeader(http.StatusOK)
	})

	return &WebServer{
		srv: &http.Server{
			Handler: mux,
			Addr:    ":8089",
		},
	}
}

func (s *WebServer) Run(ctx context.Context) {
	go func() {
		if e := s.srv.ListenAndServe(); e != http.ErrServerClosed {
			log.Println("Http server error:", e)
		}
	}()
	<-ctx.Done()

	ctx_, cancel_ := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel_()

	if e := s.srv.Shutdown(ctx_); e != nil {
		log.Println("Http server shutdown error:", e)
	}
}
*/
