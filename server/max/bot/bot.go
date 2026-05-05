package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	max "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"golang.org/x/time/rate"

	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	"github.com/jesc7/zombot/server/queue"
	"github.com/jesc7/zombot/server/types"
)

const (
	MSG_HELP = `<b>Команды бота:</b>

помощь - эта помощь
дежур[...] [кто] [дней] - дежурства
отсутств[...] - кто отсутствует
день|дни рожд[...] [дней] - ДР в ближайшие дни
`
)

var otherMessengers = []string{types.BUS_BOTTG}

type Bot struct {
	bot    *max.Api
	QWait  *queue.Queue
	chatID int64
	b      *bus.Bus
	ch     chan any
}

func NewBot(ctx context.Context, cfg types.Config, b *bus.Bus) (*Bot, error) {
	var options []max.Option
	if cfg.Proxy.Addr != "" {
		proxy, e := url.Parse(fmt.Sprintf("%s:%d", cfg.Proxy.Addr, cfg.Proxy.Port))
		if e == nil {
			options = append(options, max.WithHTTPClient(
				&http.Client{
					Transport: &http.Transport{
						Proxy: http.ProxyURL(proxy),
					},
				},
			))
		}
	}

	bot, e := max.New(cfg.Max.Token, options...)
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
		O: max.NewMessage().
			SetText(text),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) Run(ctx context.Context) {
out:
	for {
		select {
		case <-ctx.Done():
			break out

		case msg := <-b.ch: //разгребаем пакеты, пришедшие боту
			switch mt := msg.(type) {
			case shared.Envelope: //пакеты zspy

				switch mt.Type {
				//просто текст
				case shared.TypeMessageText:
					m, e := shared.Unpack[shared.MessageText](mt)
					if e != nil {
						continue
					}
					b.SendText(m.Text)
				}
			}

		case msg := <-b.QWait.Q: //разгребаем локальную очередь сообщений
			wo, ok := msg.(*queue.WaitObj)
			if !ok {
				break
			}
			m, ok := wo.O.(*max.Message)
			if !ok {
				break
			}
			b.bot.Messages.Send(ctx, m.
				SetChat(b.chatID).
				SetFormat(schemes.HTML),
			)
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
					if types.IsCommand(b.b, upd.Message.Body.Text) {
						return
					}

					if types.IsHelp(upd.Message.Body.Text) {
						upd.Message.Body.Text = "/help"
					}
					switch upd.GetCommand() {
					case "/help": //помощь
						b.SendText(MSG_HELP)
						return
					}

					//сообщения-не-команды
					for _, v := range otherMessengers {
						env, _ := shared.Pack(shared.TypeMessageText, shared.MessageText{
							Text: "<b>(Max) " + upd.Message.Sender.Name + "</b>\n" + upd.Message.Body.Text,
						})
						b.b.Write(v, env)
					}
				}()
			}
		}
	}
}
