package client

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/client/webapi"
	"github.com/jesc7/zombot/cmd/zspy/client/webskt"
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

	wg := &sync.WaitGroup{}
	skt := webskt.NewWebSocketClient(cfg)
	wg.Go(func() { //run WebSocket server
		defer func() {
			log.Println("WebSocket server has been stopped")
			cancel()
		}()
		if e := skt.Run(ctx, cfg); e != nil {
			log.Println(e)
		}
	})

	wa := webapi.NewWebServer(cfg, skt)
	wg.Go(func() { //run WebAPI server
		defer func() {
			log.Println("WebAPI server has been stopped")
			cancel()
		}()
		wa.Run(ctx)
	})

	wg.Wait()
	return ctx.Err()
}
