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

	ctypes "github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	"github.com/jesc7/zombot/server/queue"
	"github.com/jesc7/zombot/server/types"
)

var myName = types.BUS_BOTMAX
var otherMessengers = ctypes.Filter(types.ALL_MESSENGERS, func(v string) bool { return v != myName })

type Bot struct {
	bot    *maxbot.Api
	Q      *queue.Queue
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
		Q:      queue.NewQ(ctx, rate.Limit(5)),
		chatID: cfg.Max.ChatID,
		b:      b,
		ch:     b.Register(myName),
	}, e
}

func (b *Bot) SendText(ctx context.Context, core types.UniCore) {
	b.Q.Add(&queue.WaitObj{
		O: maxbot.NewMessage().
			SetText(fmt.Sprintf("%s\n%s", core.Caption, core.Text)),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) SendImage(ctx context.Context, core types.UniCore, media types.UniImage) {
	b.Q.Add(&queue.WaitObj{
		O: maxbot.NewMessage().
			SetText(fmt.Sprintf("%s\n%s\n(картинка %s (%s), размер %d)", core.Caption, core.Text, media.Name, media.Ext, len(media.Data))),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) Run(ctx context.Context) {
	go func() { //запросы в Max обрабатываем в отдельной горутине
		for {
			select {
			case <-ctx.Done():
				return

			case msg := <-b.Q.C: //разгребаем локальную очередь сообщений
				var (
					wo  *queue.WaitObj
					obj any
				)
				switch o := msg.(type) {
				case *queue.WaitObj:
					wo = o
					obj = wo.O
				case any:
					obj = o
				}

				switch mt := obj.(type) {
				case *maxbot.Message:
					b.bot.Messages.Send(ctx, mt.
						SetChat(b.chatID).
						SetFormat(schemes.HTML),
					)
				}

				if wo != nil {
					if wo.OnOk != nil {
						wo.OnOk()
					}
					wo.Done()
				}
			}
		}
	}()

out:
	for {
		select {
		case <-ctx.Done():
			break out

		case o := <-b.ch: //разгребаем пакеты, пришедшие боту по шине данных
			switch env := o.(type) {

			//пакеты zspy
			case shared.Envelope:

				switch env.Type {

				//просто текст
				case shared.TypeMessageText:
					m, e := shared.Unpack[shared.MessageText](env)
					if e != nil {
						continue
					}
					b.SendText(ctx, types.UniCore{
						Text: m.Text,
					})
				}

			// пакеты других мессенджеров или внутренние
			case types.IUniMessage:
				switch um := env.(type) {
				case types.UniMessageText:
					b.SendText(ctx, um.UniCore)

				case types.UniMessageMedia:
					switch media := um.Media[0].(type) {
					case types.UniImage:
						b.SendText(ctx, um.UniCore, media)
					}
				}
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
						types.IsCommand(b.b, myName, strconv.FormatInt(upd.GetUserID(), 10), upd.GetText()) {
						return
					}

					//остальные сообщения пересылаем в связные мессенджеры
					for _, v := range otherMessengers {
						core := types.UniCore{
							Caption: "<b><u>Max</u> | " + upd.Message.Sender.Name + "</b>",
						}

						if len(upd.Message.Body.Attachments) != 0 {
							for _, att := range upd.Message.Body.Attachments {
								log.Println(att)
							}

						} else {
							core.Text = upd.GetText()
							b.b.Write(v, types.UniMessageText{
								UniCore: core,
							})
						}
					}
				}()
			}
		}
	}
}
