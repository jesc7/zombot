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

	var mux http.ServeMux
	srv := &http.Server{Addr: ":8089"}
	srv.Handler.HandleFunc("/call", fnCalls) //пропущенные звонки
	http.HandleFunc("/zsrv", fnZsrv)         //сообщения от ZSrv
	wg.Go(func() {

	})

	wg.Wait()
	log.Println(".")
}
