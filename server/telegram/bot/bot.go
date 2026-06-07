package bot

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"

	tg "github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"golang.org/x/time/rate"

	ctypes "github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	"github.com/jesc7/zombot/server/queue"
	"github.com/jesc7/zombot/server/types"
)

var myName = types.BUS_BOTTG
var otherMessengers = ctypes.Filter(types.ALL_MESSENGERS, func(v string) bool { return v != myName })

type Bot struct {
	bot    *tg.Bot
	token  string
	me     *tg.User
	Q      *queue.Queue
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
		token:  cfg.TG.Token,
		me:     me,
		Q:      queue.NewQ(ctx, rate.Limit(5)),
		chatID: cfg.TG.ChatID,
		b:      b,
		ch:     b.Register(myName),
	}, nil
}

func (b *Bot) SendText(ctx context.Context, text string) {
	b.Q.Add(&queue.WaitObj{
		O: tu.Message(tu.ID(b.chatID), text),
	}, queue.PRIORITY_NORMAL)
}

func R[T *tg.ForumTopic | *tg.Message | *tg.File](a []any, i int) (res T) {
	if len(a) > i && a[i] != nil {
		switch v := a[i].(type) {
		case *tg.ForumTopic:
			res = any(v).(T)
		case *tg.Message:
			res = any(v).(T)
		case *tg.File:
			res = any(v).(T)
		}
	}
	return
}

func E(a []any, i int) error {
	if len(a) > i && a[i] != nil {
		return a[i].(error)
	}
	return nil
}

func (bot *Bot) GetFile(ctx context.Context, fileID string) ([]byte, string, error) {
	var (
		e error
		f *tg.File
	)
	bot.Q.Wait(ctx, &queue.WaitObj{
		O: &tg.GetFileParams{
			FileID: fileID,
		},
		OnOk: func(a ...any) { f, e = R[*tg.File](a, 0), E(a, 1) },
	}, queue.PRIORITY_NORMAL)
	if e != nil {
		return []byte{}, "", e
	}

	resp, e := ctypes.GetWithContext(ctx, nil, "https://api.telegram.org/file/bot"+bot.token+"/"+f.FilePath)
	if e != nil {
		return []byte{}, "", e
	}
	defer resp.Body.Close()

	b := new(bytes.Buffer)
	if _, e = b.ReadFrom(resp.Body); e != nil {
		return []byte{}, "", e
	}
	return b.Bytes(), mime.TypeByExtension(filepath.Ext(f.FilePath)), nil
}

func (b *Bot) Run(ctx context.Context) error {
	updates, e := b.bot.UpdatesViaLongPolling(ctx, &tg.GetUpdatesParams{
		Offset:  -1,
		Limit:   0,
		Timeout: 10,
		AllowedUpdates: []string{
			tg.MessageUpdates,
			//tg.EditedMessageUpdates,
			//tg.CallbackQueryUpdates,
			//tg.MessageReactionUpdates,
		},
	})
	if e != nil {
		return e
	}
	defer b.bot.StopPoll(ctx, nil)

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
					b.SendText(ctx, m.Text)
				}

			// пакеты других мессенджеров или внутренние
			case types.IUniMessage:
				switch um := env.(type) {
				case types.UniMessageText:
					b.SendText(ctx, um.Text)
				}
			}

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
			case *tg.SendMessageParams:
				b.bot.SendMessage(ctx, mt.
					WithParseMode(tg.ModeHTML),
				)

			case *tg.GetFileParams:
				var r *tg.File
				if r, e = b.bot.GetFile(ctx, mt); wo != nil && wo.OnOk != nil {
					wo.OnOk(r, e)
				}
			}

			if wo != nil {
				wo.Done()
			}

		case update := <-updates: //приехали апдейты с сервера
			func() {
				msg := update.Message
				if msg == nil {
					return
				}
				//только групповой чат из настроек
				if msg.Chat.ID != b.chatID {
					return
				}
				//отсеиваем команды
				if types.IsCommand(b.b, myName, strconv.FormatInt(msg.From.ID, 10), update.Message.Text) {
					return
				}

				//остальные сообщения пересылаем в связные мессенджеры
				for _, v := range otherMessengers {
					caption := "<b><u>Telegram</u> | " + msg.From.FirstName + "</b>\n"

					if msg.Photo != nil {
						file, _, e := b.GetFile(ctx, msg.Photo[len(msg.Photo)-1].FileID)
						if e != nil {
							return
						}

						var media []types.IUniMedia
						media = append(media, types.UniImage{
							UniMedia: types.UniMedia{
								Caption: msg.Caption,
								Data:    file,
							},
						},
						)

						b.b.Write(v, types.UniMessageMedia{
							Media: []types.UniImage{
								{
									Caption: msg.Caption,
									Data:    file,
								},
							},
							//Text:  caption + update.Message.Text,
						})
					} else if msg.Audio != nil {

					} else if msg.Voice != nil {

					} else if msg.Video != nil {

					} else if msg.VideoNote != nil {

					} else if msg.Document != nil {

					} else {
						b.b.Write(v, types.UniMessageText{
							Text: caption + update.Message.Text,
						})
					}
				}
			}()
		}
	}
	return nil
}
