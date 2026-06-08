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
	"strings"
	"time"

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
	proxy  string
	me     *tg.User
	Q      *queue.Queue
	chatID int64
	b      *bus.Bus
	ch     chan any
}

func NewBot(ctx context.Context, cfg types.Config, b *bus.Bus) (*Bot, error) {
	var proxyAddr string
	options := append([]tg.BotOption{}, tg.WithDefaultLogger(false, true))
	if cfg.Proxy.Addr != "" {
		proxyAddr = fmt.Sprintf("%s:%d", cfg.Proxy.Addr, cfg.Proxy.Port)
		proxy, e := url.Parse(proxyAddr)
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
		proxy:  proxyAddr,
		me:     me,
		Q:      queue.NewQ(ctx, rate.Limit(5)),
		chatID: cfg.TG.ChatID,
		b:      b,
		ch:     b.Register(myName),
	}, nil
}

func (b *Bot) GetFile(ctx context.Context, fileID string) ([]byte, string, error) {
	var (
		e error
		f *tg.File
	)
	b.Q.Wait(ctx, &queue.WaitObj{
		O: &tg.GetFileParams{
			FileID: fileID,
		},
		OnOk: func(a ...any) {
			defer recover()
			if a[0] != nil {
				f = a[0].(*tg.File)
			}
			if a[1] != nil {
				e = a[1].(error)
			}
		},
	}, queue.PRIORITY_NORMAL)
	if e != nil {
		return []byte{}, "", e
	}

	c := &http.Client{Timeout: 10 * time.Second}
	if b.proxy != "" {
		proxy, _ := url.Parse(b.proxy)
		c.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	}
	resp, e := ctypes.GetWithContext(ctx, c, "https://api.telegram.org/file/bot"+b.token+"/"+f.FilePath)
	if e != nil {
		return []byte{}, "", e
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	if _, e = buf.ReadFrom(resp.Body); e != nil {
		return []byte{}, "", e
	}
	return buf.Bytes(), mime.TypeByExtension(filepath.Ext(f.FilePath)), nil
}

func (b *Bot) SendText(ctx context.Context, core types.UniCore) {
	text, entities := tu.MessageEntities(([]tu.MessageEntityCollection{
		tu.Entity(core.Caption + "\n"),
		tu.Entity(core.Text),
	})...)
	b.Q.Add(&queue.WaitObj{
		O: tu.Message(tu.ID(b.chatID), text).
			WithEntities(entities...),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) SendImage(ctx context.Context, core types.UniCore, media types.UniImage) {
	text, entities := tu.MessageEntities(([]tu.MessageEntityCollection{
		tu.Entity(core.Caption + "\n"),
		tu.Entity(core.Text),
	})...)

	//tu.MediaGroup()
	b.Q.Add(&queue.WaitObj{
		O: tu.Photo(tu.ID(b.chatID), tu.FileFromBytes(media.Data, strings.Replace(time.Now().Format("Image_150405.000000"), ".", "", 1))).
			WithCaption(text).
			WithCaptionEntities(entities...),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) SendMediaGroup(ctx context.Context, core types.UniCore, mediaGroup []tg.InputMedia) {
	b.Q.Add(&queue.WaitObj{
		O: tu.MediaGroup(tu.ID(b.chatID), mediaGroup...),
	}, queue.PRIORITY_NORMAL)
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

	go func() { //запросы в Telegram обрабатываем в отдельной горутине
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
				case *tg.SendMessageParams:
					b.bot.SendMessage(ctx, mt.
						WithParseMode(tg.ModeHTML),
					)

				case *tg.GetFileParams:
					var r *tg.File
					if r, e = b.bot.GetFile(ctx, mt); wo != nil && wo.OnOk != nil {
						wo.OnOk(r, e)
					}

				case *tg.SendPhotoParams:
					var r *tg.Message
					if r, e = b.bot.SendPhoto(ctx, mt.WithParseMode(tg.ModeHTML)); wo != nil && wo.OnOk != nil {
						wo.OnOk(r, e)
					}

				case *tg.SendMediaGroupParams:
					if _, e := b.bot.SendMediaGroup(ctx, mt); wo != nil && wo.OnOk != nil {
						wo.OnOk(e)
					}
				}

				if wo != nil {
					wo.Done()
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

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
				switch msg := env.(type) {
				case types.UniMessageText:
					b.SendText(ctx, msg.UniCore)

				case types.UniMessageMedia:
					if msg.IsCollage() && len(msg.Media) < 11 {
						var mediaGroup []tg.InputMedia
						if msg.IsPhotoCollage() {
							text, entities := tu.MessageEntities(([]tu.MessageEntityCollection{
								tu.Entity(msg.Caption + "\n"),
								tu.Entity(msg.Text),
							})...)
							for i, media := range msg.Media {
								photo := &tg.InputMediaPhoto{
									Type:            tg.MediaTypePhoto,
									Media:           tu.FileFromBytes(media.(types.UniImage).Data, strings.Replace(time.Now().Format("Image_150405.000000"), ".", "", 1)),
									Caption:         text,
									CaptionEntities: entities,
									ParseMode:       tg.ModeHTML,
								}
								if i != 0 {
									photo.Caption = ""
									photo.CaptionEntities = []tg.MessageEntity{}
								}
								mediaGroup = append(mediaGroup, photo)
							}
						}
						b.SendMediaGroup(ctx, msg.UniCore, mediaGroup)

					} else {
						var core types.UniCore
						for i, media := range msg.Media {
							if i == 0 {
								core = msg.UniCore
							}
							switch mt := media.(type) {
							case types.UniImage:
								b.SendImage(ctx, core, mt)
							}
							core.Text = "" //текст сообщения только у первого файла
						}
					}
				}
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
					core := types.UniCore{
						Caption: "<b><u>Telegram</u> | " + msg.From.FirstName + "</b>",
					}

					if msg.Photo != nil {
						file, ext, e := b.GetFile(ctx, msg.Photo[len(msg.Photo)-1].FileID)
						if e != nil {
							return
						}

						var media types.UniImage
						media.Data = file
						media.Ext = ext
						core.Text = msg.Caption

						b.b.Write(v, types.UniMessageMedia{
							UniCore: core,
							Media:   append([]types.IUniMedia{}, media),
						})

					} else if msg.Audio != nil {

					} else if msg.Voice != nil {

					} else if msg.Video != nil {

					} else if msg.VideoNote != nil {

					} else if msg.Document != nil {

					} else {
						core.Text = update.Message.Text
						b.b.Write(v, types.UniMessageText{
							UniCore: core,
						})
					}
				}
			}()
		}
	}
	//return nil
}
