package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	tg "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"golang.org/x/time/rate"

	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	"github.com/jesc7/zombot/server/queue"
	"github.com/jesc7/zombot/server/types"
)

type Bot struct {
	bot    *tg.Bot
	me     *tg.User
	QWait  *queue.Queue
	chatID int64
	b      *bus.Bus
	ch     chan any
}

func NewBot(ctx context.Context, cfg types.Config, b *bus.Bus) (*Bot, error) {
	options := append([]tg.BotOption{}, tg.WithDefaultLogger(false, true))
	if cfg.Proxy.Addr != "" {
		proxy, e := url.Parse(fmt.Sprintf("%s:%d", cfg.Proxy.Addr, cfg.Proxy.Port))
		if e == nil {
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
	if e != nil {
		return nil, e
	}
	me, e := bot.GetMe(ctx)
	if e != nil {
		return nil, e
	}
	return &Bot{
		bot:    bot,
		me:     me,
		QWait:  queue.NewQ(ctx, rate.Limit(5)),
		chatID: cfg.TG.ChatID,
		b:      b,
		ch:     b.Register(types.BUS_BOTTG),
	}, nil
}

func (b *Bot) SendText(text string) {
	b.QWait.Add(&queue.WaitObj{
		O: tu.Message(tu.ID(b.chatID), text),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) Run(ctx context.Context) error {
	updates, e := b.bot.UpdatesViaLongPolling(ctx, &tg.GetUpdatesParams{
		Offset:  -1,
		Limit:   0,
		Timeout: 10,
		AllowedUpdates: []string{
			tg.MessageUpdates,
			tg.EditedMessageUpdates,
			tg.CallbackQueryUpdates,
			tg.MessageReactionUpdates,
		},
	})
	if e != nil {
		return e
	}
	defer b.bot.StopPoll(ctx, nil)

	go func() {
	out:
		for {
			select {
			case <-ctx.Done():
				break out

			case msg := <-b.ch: //разгребаем пакеты, пришедшие боту
				switch mt := msg.(type) {
				case shared.Envelope: //пакеты zspy
					log.Println("Bot", mt.Type)

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
				switch mt := wo.O.(type) {
				case *tg.SendMessageParams:
					b.bot.SendMessage(ctx, mt.
						WithParseMode(tg.ModeHTML),
					)
				}
				if wo.OnOk != nil {
					wo.OnOk()
				}
			}
		}
	}()

	bh, e := th.NewBotHandler(b.bot, updates)
	if e != nil {
		return e
	}
	defer bh.Stop()
	//do work

	return bh.Start()
}
