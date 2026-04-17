package bot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jesc7/zombot/types"
	max "github.com/max-messenger/max-bot-api-client-go"
)

type TextMsg struct {
	Text string
}
type Bot struct {
	bot    *max.Api
	income chan *max.Message
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
	bot, e := max.New(cfg.Max.Token, options...)
	return &Bot{
		bot: bot,
	}, e
}

func (b *Bot) SendText(text string) {
	b.income <- max.NewMessage().SetText(text)
}

func (b *Bot) SendCall(phone string) {
	b.income <- max.NewMessage().SetText(fmt.Sprintf("📞 Вам звонили%s: <b>%s</b>\n", types.Iif(strings.HasPrefix(phone, "8800 "), " на 8800", ""), phone))
}

func (b *Bot) SendZSrv(msg types.ZSrvMessage) {
	//b.income <- max.NewMessage().SetText(fmt.Sprintf("📞 Вам звонили%s: <b>%s</b>\n", types.Iif(strings.HasPrefix(phone, "8800 "), " на 8800", ""), phone))
}

func (b *Bot) Run(ctx context.Context) {
out:
	for {
		select {
		case <-ctx.Done():
			break out

		case msg := <-b.income:
			_ = msg

		case upd := <-b.bot.GetUpdates(ctx):
		}
	}
	b.bot.GetUpdates()
}
