package server

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	max_bot "github.com/jesc7/zombot/server/max/bot"
	tg_bot "github.com/jesc7/zombot/server/telegram/bot"
	"github.com/jesc7/zombot/server/types"
	"github.com/jesc7/zombot/server/ws"
)

func Start(ctx context.Context, service bool) error {
	bin, e := runPath(service)
	if e != nil {
		return e
	}

	f, e := os.ReadFile(filepath.Join(filepath.Dir(bin), "cfg.json"))
	if e != nil {
		return e
	}
	var cfg types.Config
	if e = json.Unmarshal(f, &cfg); e != nil {
		return e
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	myBus := bus.NewBus()
	defer myBus.Close()

	srv, e := ws.NewWebSocketServer(ctx, cfg, myBus)
	botMax, e := max_bot.NewBot(ctx, cfg, myBus)
	if e != nil {
		log.Fatalln("Can't create Max bot:", e)
	}

	botTG, e := tg_bot.NewBot(ctx, cfg, myBus)
	if e != nil {
		log.Fatalln("Can't create Telegram bot:", e)
	}

	wg := &sync.WaitGroup{}
	wg.Go(func() { //run WebSocket server
		defer cancel()
		if e = srv.Run(ctx); e != nil {
			log.Println(e)
		}
	})

	wg.Go(func() { //run Max bot
		defer cancel()
		botMax.Run(ctx)
	})

	wg.Go(func() { //run Telegram bot
		defer cancel()
		if e = botTG.Run(ctx); e != nil {
			log.Println(e)
		}
	})

	wg.Wait()
	return ctx.Err()
}
