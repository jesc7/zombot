package server

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	maxbot "github.com/jesc7/zombot/server/max/bot"
	"github.com/jesc7/zombot/server/types"
	"github.com/jesc7/zombot/server/ws"
)

func Start(ctx context.Context, service bool) error {
	cwd, e := runPath(service)
	if e != nil {
		return e
	}

	f, e := os.ReadFile(filepath.Join(filepath.Dir(cwd), "cfg.json"))
	if e != nil {
		return e
	}
	var cfg types.Config
	if e = json.Unmarshal(f, &cfg); e != nil {
		return e
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	srv := ws.NewWS(cfg)
	bot, e := maxbot.NewBot(ctx, cfg, srv)
	if e != nil {
		log.Fatalln("Can't create Max bot:", e)
	}

	wg := &sync.WaitGroup{}
	wg.Go(func() { //run WebSocket server
		defer func() {
			log.Println("WebSocket server has been stopped")
			cancel()
		}()
		srv.Run(ctx)
	})

	wg.Go(func() { //run Max bot
		defer func() {
			log.Println("Max bot has been stopped")
			bot.Free()
			cancel()
		}()
		bot.Run()
	})

	wg.Wait()
	log.Println(".")
	return ctx.Err()
}
