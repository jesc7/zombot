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

	wg := &sync.WaitGroup{}
	skt := webskt.NewWebSocketClient(cfg)
	wg.Go(func() { //run WebSocket server
		defer func() {
			log.Println("WebSocket server has been stopped")
			cancel()
		}()
		skt.Run(ctx)
	})

	wa := webapi.NewWebServer()
	wg.Go(func() { //run WebAPI server
		defer func() {
			log.Println("WebAPI server has been stopped")
			cancel()
		}()
		wa.Run(ctx, skt)
	})

	wg.Wait()
	return ctx.Err()
}
