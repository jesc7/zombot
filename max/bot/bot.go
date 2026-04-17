package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/jesc7/zombot/types"
	max "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

type TextMsg struct {
	Text string
}
type Bot struct {
	bot    *max.Api
	income chan *max.Message
	chatID int64
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
		bot:    bot,
		chatID: cfg.Max.ChatID,
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
			switch ut := upd.(type) {
			case *schemes.MessageCreatedUpdate:
				m := ut.Message
				if m.Recipient.ChatType != schemes.CHAT || m.Recipient.ChatId != b.chatID {
					break
				}

				if e := b.bot.Messages.Send(ctx, max.NewMessage().
					SetChat(m.Recipient.ChatId).
					SetText(m.Body.Text)); e != nil {
					log.Println("Send message error:", e)
				}
			}
		}
	}
}
