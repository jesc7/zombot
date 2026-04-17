package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	maxBot "github.com/jesc7/zombot/max/bot"
	"github.com/jesc7/zombot/types"
)

func main() {
	f, e := os.ReadFile(filepath.Join(filepath.Dir(os.Args[0]), "cfg.json"))
	if e != nil {
		log.Fatalln("Can't read config file:", e)
	}
	var cfg types.Config
	if e = json.Unmarshal(f, &cfg); e != nil {
		log.Fatalln("Can't unmarshal the json:", e)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	bot, e := maxBot.NewBot(ctx, cfg)
	if e != nil {
		log.Fatalln("Can't create Max bot:", e)
	}

	wg := &sync.WaitGroup{}

	//run Max bot
	wg.Go(func() {
		defer cancel()
		bot.Run()
	})

	//run http server
	//пропущенные звонки http-сервер принимает на порту :8089
	//скрипт asterisk 192.168.67.11/etc/asterisk/IgorBot.php шлет запрос вида 'ip:8089/call?phone=XXXXXX'
	chCalls := make(chan string)
	defer close(chCalls)
	fnCalls := func(w http.ResponseWriter, r *http.Request) {
		if v, ok := r.URL.Query()["phone"]; ok {
			chCalls <- v[0]
		}
	}

	//различные сообщения от ZSrv, например, не обновляются прайсы, долго нет заказов и т.д. формат: 'ip:8089/zsrv', body json
	chZSrv := make(chan types.ZSrvMessage)
	defer close(chZSrv)
	fnZSrv := func(w http.ResponseWriter, r *http.Request) {
		var msg types.ZSrvMessage
		if e := json.NewDecoder(r.Body).Decode(&msg); e != nil {
			http.Error(w, e.Error(), http.StatusBadRequest)
			return
		}
		chZSrv <- msg
		w.WriteHeader(http.StatusOK)
	}

	mux := &http.ServeMux{}
	mux.HandleFunc("/call", fnCalls) //пропущенные звонки
	mux.HandleFunc("/zsrv", fnZSrv)  //сообщения от ZSrv
	srv := &http.Server{
		Handler: mux,
		Addr:    ":8089",
	}
	wg.Go(func() {
		go func() {
			defer cancel()

			if e := srv.ListenAndServe(); e != http.ErrServerClosed {
				log.Println("Http server error:", e)
			}
		}()
		<-ctx.Done()

		srvCtx, srvCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer srvCancel()

		if e := srv.Shutdown(srvCtx); e != nil {
			log.Println("Http server shutdown error:", e)
		}
	})

	wg.Go(func() {
	out:
		for {
			select {
			case <-ctx.Done():
				break out

			case call := <-chCalls:
				bot.SendText(fmt.Sprintf("📞 Вам звонили%s: <b>%s</b>\n", types.Iif(strings.HasPrefix(call, "8800 "), " на 8800", ""), call))

			case msgZSrv := <-chZSrv:
				_ = msgZSrv
			}
		}
	})

	wg.Wait()
	log.Println(".")
}
