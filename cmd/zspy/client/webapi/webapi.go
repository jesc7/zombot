package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jesc7/zombot/cmd/zspy/client/phones"
	"github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/client/webskt"
	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type WebServer struct {
	srv *http.Server
	skt *webskt.WebSocketClient
}

var (
	reCall = regexp.MustCompile(`(?:\+?\d[\-\s]?\(?\s?\d{3,5}\s?\)?[\-\s]?)?(?:\d[\-\s]?){4,6}\d`)
)

func NewWebServer(cfg types.Config, cwd string, skt *webskt.WebSocketClient) *WebServer {
	mux := http.NewServeMux()
	//скрипт asterisk 192.168.67.11/etc/asterisk/IgorBot.php шлет запрос вида 'ip:8089/call?phone=XXXXXX'
	mux.HandleFunc("/call", func(w http.ResponseWriter, r *http.Request) {
		v, ok := r.URL.Query()["phone"]
		if !ok {
			return
		}
		phone := reCall.FindAllString(v[0], -1)
		if len(phone) == 0 {
			return
		}

		env, _ := shared.Pack(shared.TypeMessageCall, shared.MessageCall{
			Prefix: types.Iif(strings.HasPrefix(phone[0], "8800 "), "8800", ""),
			Phone:  phone[0],
			Region: phones.FindByPhone(cwd, phone[0]),
		})
		skt.Write(env)
	})

	//сообщения от ZSrv
	mux.HandleFunc("/zsrv", func(w http.ResponseWriter, r *http.Request) {
		var msg shared.MessageZSRV
		if e := json.NewDecoder(r.Body).Decode(&msg); e != nil {
			http.Error(w, e.Error(), http.StatusBadRequest)
			return
		}
		env, _ := shared.Pack(shared.TypeMessageZSRV, msg)
		skt.Write(env)
		w.WriteHeader(http.StatusOK)
	})

	return &WebServer{
		srv: &http.Server{
			Handler: mux,
			Addr:    fmt.Sprintf(":%d", cfg.WA.Port), //8089
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
