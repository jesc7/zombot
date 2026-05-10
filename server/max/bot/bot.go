package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"golang.org/x/time/rate"

	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	"github.com/jesc7/zombot/server/queue"
	"github.com/jesc7/zombot/server/types"
)

var otherMessengers = []string{types.BUS_BOTTG}

type Bot struct {
	bot    *maxbot.Api
	QWait  *queue.Queue
	chatID int64
	b      *bus.Bus
	ch     chan any
}

func NewBot(ctx context.Context, cfg types.Config, b *bus.Bus) (*Bot, error) {
	var options []maxbot.Option
	if cfg.Proxy.Addr != "" {
		proxy, e := url.Parse(fmt.Sprintf("%s:%d", cfg.Proxy.Addr, cfg.Proxy.Port))
		if e == nil {
			options = append(options, maxbot.WithHTTPClient(
				&http.Client{
					Transport: &http.Transport{
						Proxy: http.ProxyURL(proxy),
					},
				},
			))
		}
	}

	bot, e := maxbot.New(cfg.Max.Token, options...)
	return &Bot{
		bot:    bot,
		QWait:  queue.NewQ(ctx, rate.Limit(5)),
		chatID: cfg.Max.ChatID,
		b:      b,
		ch:     b.Register(types.BUS_BOTMAX),
	}, e
}

func (b *Bot) SendText(text string) {
	b.QWait.Add(&queue.WaitObj{
		O: maxbot.NewMessage().
			SetText(text),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) Run(ctx context.Context) {
out:
	for {
		select {
		case <-ctx.Done():
			break out

		case o := <-b.ch: //разгребаем пакеты, пришедшие боту
			switch env := o.(type) {
			case shared.Envelope: //пакеты zspy

				switch env.Type {
				//просто текст
				case shared.TypeMessageText:
					m, e := shared.Unpack[shared.MessageText](env)
					if e != nil {
						continue
					}
					b.SendText(m.Text)
				}
			}

		case o := <-b.QWait.Q: //разгребаем локальную очередь сообщений
			wo, ok := o.(*queue.WaitObj)
			if !ok {
				break
			}
			switch msg := wo.O.(type) {
			case *maxbot.Message:
				b.bot.Messages.Send(ctx, msg.
					SetChat(b.chatID).
					SetFormat(schemes.HTML),
				)
			}
			if wo.OnOk != nil {
				wo.OnOk()
			}

		case update := <-b.bot.GetUpdates(ctx): //приехали апдейты с сервера
			switch upd := update.(type) {
			case *schemes.MessageCreatedUpdate:
				func() {
					log.Println("Message from", upd.GetChatID())

					//только групповой чат из настроек
					if upd.Message.Recipient.ChatType != schemes.CHAT || upd.GetChatID() != b.chatID {
						return
					}

					//отсеиваем команды
					if len(upd.Message.Body.Attachments) == 0 &&
						types.IsCommand(b.b, types.BUS_BOTMAX, strconv.FormatInt(upd.GetUserID(), 10), upd.GetText()) {
						return
					}

					//сообщения-не-команды
					for _, v := range otherMessengers {
						b.b.Write(v, types.UniMessage(&types.UniMessageText{
							Text: "<b><u>Max</u> | " + upd.Message.Sender.Name + "</b>\n" + upd.GetText(),
						}))
					}
				}()
			}
		}
	}
}
