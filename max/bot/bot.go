package bot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jesc7/zombot/types"
	max "github.com/max-messenger/max-bot-api-client-go"
)

type Bot struct {
	bot *max.Api
	ctx context.Context
}

func NewBot(ctx context.Context, cfg types.Config) (*Bot, error) {
	var options []max.Option
	if cfg.Proxy.Addr != "" {
		proxy, e := url.Parse(fmt.Sprintf("%s:%d", cfg.Proxy.Addr, cfg.Proxy.Port))
		if e != nil {
			return nil, e
		}

		options = append(options, max.WithHTTPClient(
			&http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxy),
				},
			},
		))
	}
	b, e := max.New(cfg.Max.Token, options...)
	return &Bot{
		ctx: ctx,
		bot: b,
	}, e
}
