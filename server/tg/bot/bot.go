package bot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	tg "github.com/mymmrac/telego"
	"golang.org/x/time/rate"

	//th "github.com/mymmrac/telego/telegohandler"
	//tu "github.com/mymmrac/telego/telegoutil"

	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	"github.com/jesc7/zombot/server/queue"
	"github.com/jesc7/zombot/server/types"
)

type Bot struct {
	bot    *tg.Bot
	QWait  *queue.Queue
	chatID int64
	b      *bus.Bus
	ch     chan shared.Envelope
}

func NewBot(ctx context.Context, cfg types.Config, b *bus.Bus) (*Bot, error) {
	ch, e := b.Register(types.BUS_BOTTG)
	if e != nil {
		return nil, e
	}

	options := append([]tg.BotOption{}, tg.WithDefaultLogger(false, true))
	if cfg.Proxy.Addr != "" {
		var proxy *url.URL
		if proxy, e = url.Parse(fmt.Sprintf("%s:%d", cfg.Proxy.Addr, cfg.Proxy.Port)); e == nil {
			options = append(options, tg.WithHTTPClient(
				&http.Client{
					Transport: &http.Transport{
						Proxy: http.ProxyURL(proxy),
					},
				},
			))
		}
	}

	bot, e := tg.NewBot(cfg.TG.Token, options...)
	return &Bot{
		bot:    bot,
		QWait:  queue.NewQ(ctx, rate.Limit(5)),
		chatID: cfg.TG.ChatID,
		b:      b,
		ch:     ch,
	}, e
}
