package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	maxbot "github.com/jesc7/zombot/max/bot"
	"github.com/jesc7/zombot/types"
	"github.com/jesc7/zombot/webapi"
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

	bot, e := maxbot.NewBot(ctx, cfg)
	if e != nil {
		log.Fatalln("Can't create Max bot:", e)
	}

	wg := &sync.WaitGroup{}

	//run Max bot
	wg.Go(func() {
		defer func() {
			log.Println("Max bot has been stopped")
			bot.Free()
			cancel()
		}()
		bot.Run(ctx)
	})

	//run WebServer
	server := webapi.NewServer(ctx, cfg, bot)
	wg.Go(func() {
		defer func() {
			log.Println("WebServer has been stopped")
			cancel()
		}()
		server.Run(ctx)
	})

	wg.Wait()
	log.Println(".")
}
