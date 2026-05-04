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

	myBus := bus.NewBus() //data bus
	defer myBus.Close()

	wg := &sync.WaitGroup{}

	//WebSocket server
	wsServer, e := ws.NewWebSocketServer(ctx, cfg, myBus)
	if e != nil {
		log.Fatalln("Can't create WebSocket server:", e)
	}
	wg.Go(func() { //run server
		defer cancel()
		if e = wsServer.Run(ctx); e != nil {
			log.Println(e)
		}
	})

	//Max bot
	botMax, e := max_bot.NewBot(ctx, cfg, myBus)
	if e != nil {
		log.Println("Can't create Max bot:", e)
	} else {
		wg.Go(func() { //run bot
			defer func() {
				myBus.Unregister(types.BUS_BOTMAX)
				cancel()
			}()
			botMax.Run(ctx)
		})
	}

	//Telegram bot
	botTelegram, e := tg_bot.NewBot(ctx, cfg, myBus)
	if e != nil {
		log.Println("Can't create Telegram bot:", e)
	} else {
		wg.Go(func() { //run bot
			defer func() {
				myBus.Unregister(types.BUS_BOTTG)
				cancel()
			}()
			if e = botTelegram.Run(ctx); e != nil {
				log.Println(e)
			}
		})
	}

	wg.Wait()
	return ctx.Err()
}
