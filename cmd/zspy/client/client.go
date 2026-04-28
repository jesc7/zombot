package client

import (
	"context"
	"database/sql"
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

	db, e := sql.Open(cfg.DB.Driver, cfg.DB.ConnStr)
	if e != nil {
		return e
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}
	skt := webskt.NewWebSocketClient(cfg)
	wg.Go(func() { //run WebSocket server
		defer func() {
			log.Println("WebSocket server has been stopped")
			cancel()
		}()
		skt.Run(ctx, db)
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
