package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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

	//Run Max bot
	wg.Go(func() {
		bot.Run()
	})

	mux := &http.ServeMux{}
	mux.HandleFunc("/call", fnCalls) //пропущенные звонки
	mux.HandleFunc("/zsrv", fnZsrv)  //сообщения от ZSrv
	srv := &http.Server{
		Handler: mux,
		Addr:    ":8089",
	}
	wg.Go(func() {
		go func() {
			defer cancel()

			if err := srv.ListenAndServe(); e != http.ErrServerClosed {
				log.Fatalf("ListenAndServe error: %v", err)
			}
		}()
		<-ctx.Done()

		shutdownCtx, forceCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer forceCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("Ошибка при закрытии сервера: %v", err)
		}
	})

	wg.Wait()
	log.Println(".")
}
