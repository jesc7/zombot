package bot

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

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
	token  string
	proxy  string
	Q      *queue.Queue
	chatID int64
	b      *bus.Bus
	ch     chan any
}

func NewBot(ctx context.Context, cfg types.Config, b *bus.Bus) (*Bot, error) {
	var proxyAddr string
	var options []maxbot.Option
	if cfg.Proxy.Addr != "" {
		proxyAddr = fmt.Sprintf("%s:%d", cfg.Proxy.Addr, cfg.Proxy.Port)
		proxy, e := url.Parse(proxyAddr)
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
		token:  cfg.Max.Token,
		proxy:  proxyAddr,
		Q:      queue.NewQ(ctx, rate.Limit(5)),
		chatID: cfg.Max.ChatID,
		b:      b,
		ch:     b.Register(myName),
	}, e
}

func (b *Bot) GetFile(ctx context.Context, fileUrl string) ([]byte, string, error) {
	c := &http.Client{Timeout: 10 * time.Second}
	if b.proxy != "" {
		proxy, _ := url.Parse(b.proxy)
		c.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	}
	resp, e := ctypes.GetWithContext(ctx, c, fileUrl)
	if e != nil {
		return []byte{}, "", e
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	if _, e = buf.ReadFrom(resp.Body); e != nil {
		return []byte{}, "", e
	}
	return buf.Bytes(), "", nil
}

func (b *Bot) SendText(ctx context.Context, core types.UniCore) {
	b.Q.Add(&queue.WaitObj{
		O: maxbot.NewMessage().
			SetText(fmt.Sprintf("%s\n%s", core.Caption, core.Text)),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) SendImage(ctx context.Context, core types.UniCore, media types.UniImage) {
	photo, e := b.bot.Uploads.UploadPhotoFromReader(ctx, bytes.NewReader(media.Data))
	if e != nil {
		return
	}

	b.Q.Add(&queue.WaitObj{
		O: maxbot.NewMessage().
			SetText(fmt.Sprintf("%s\n%s", core.Caption, core.Text)).
			AddPhoto(photo),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) SendVideo(ctx context.Context, core types.UniCore, media types.UniVideo) {
	video, e := b.bot.Uploads.UploadMediaFromReader(ctx, schemes.VIDEO, bytes.NewReader(media.Data))
	if e != nil {
		return
	}

	b.Q.Add(&queue.WaitObj{
		O: maxbot.NewMessage().
			SetText(fmt.Sprintf("%s\n%s", core.Caption, core.Text)).
			AddVideo(video),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) SendAudio(ctx context.Context, core types.UniCore, media types.UniAudio) {
	audio, e := b.bot.Uploads.UploadMediaFromReader(ctx, schemes.AUDIO, bytes.NewReader(media.Data))
	if e != nil {
		return
	}

	b.Q.Add(&queue.WaitObj{
		O: maxbot.NewMessage().
			SetText(fmt.Sprintf("%s\n%s", core.Caption, core.Text)).
			AddAudio(audio),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) SendDocument(ctx context.Context, core types.UniCore, media types.UniDocument) {
	file, e := b.bot.Uploads.UploadMediaFromReader(ctx, schemes.FILE, bytes.NewReader(media.Data))
	if e != nil {
		return
	}

	b.Q.Add(&queue.WaitObj{
		O: maxbot.NewMessage().
			SetText(fmt.Sprintf("%s\n%s", core.Caption, core.Text)).
			AddFile(file),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) SendContacts(ctx context.Context, msg types.UniMessageContacts) {
	text := msg.Caption + " отправил(а) контакты 📞\n"
	for _, c := range msg.Contacts {
		text += "<b>" + c.Caption + "</b>\n"
		coll = append(coll, tu.Entity("<b>"+c.Caption+"</b>\n"))
		ent := tu.Entity(c.Phone + "\n")
		if rePhone.MatchString(c.Phone) {
			ent = ent.PhoneNumber()
		}
		coll = append(coll, ent)
		coll = append(coll, tu.Entity("\n"))
	}
	b.Q.Add(&queue.WaitObj{
		O: tu.MessageWithEntities(tu.ID(b.chatID), coll...),
	}, queue.PRIORITY_NORMAL)
	return
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
				switch msg := env.(type) {
				case types.UniMessageText:
					b.SendText(ctx, msg.UniCore)

				case types.UniMessageMedia:
					var core types.UniCore
					for i, media := range msg.Media {
						if i == 0 {
							core = msg.UniCore
						}
						switch mt := media.(type) {
						case types.UniImage:
							b.SendImage(ctx, core, mt)

						case types.UniVideo:
							b.SendVideo(ctx, core, mt)

						case types.UniAudio:
							b.SendAudio(ctx, core, mt)

						case types.UniDocument:
							b.SendDocument(ctx, core, mt)
						}
						core.Text = "" //текст сообщения только у первого файла
					}

				case types.UniMessageContacts:
					b.SendContacts(ctx, msg)
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
							core.Text = upd.GetText()
							media := types.UniMessageMedia{UniCore: core}
							contacts := types.UniMessageContacts{UniCore: core}

							for _, attach := range upd.Message.Body.Attachments {
								switch at := attach.(type) {
								case *schemes.PhotoAttachment:
									file, _, e := b.GetFile(ctx, at.Payload.Url)
									if e != nil {
										return
									}

									media.Media = append(media.Media, types.UniImage{
										UniMedia: types.UniMedia{
											Data: file,
										},
									})

								case *schemes.VideoAttachment:
									file, _, e := b.GetFile(ctx, at.Payload.Url)
									if e != nil {
										return
									}

									media.Media = append(media.Media, types.UniVideo{
										UniMedia: types.UniMedia{
											Data: file,
										},
									})

								case *schemes.AudioAttachment:
									file, _, e := b.GetFile(ctx, at.Payload.Url)
									if e != nil {
										return
									}

									media.Media = append(media.Media, types.UniAudio{
										UniMedia: types.UniMedia{
											Data: file,
										},
									})

								case *schemes.FileAttachment:
									file, _, e := b.GetFile(ctx, at.Payload.Url)
									if e != nil {
										return
									}

									media.Media = append(media.Media, types.UniDocument{
										UniMedia: types.UniMedia{
											Data: file,
											Name: at.Filename,
										},
									})

								case *schemes.ContactAttachment:
									contacts.Contacts = append(contacts.Contacts, types.Contact{
										Caption: at.Payload.TamInfo.FirstName,
										Phone:   "[скрыт]",
									})
								}
							}
							switch {
							case len(media.Media) != 0:
								b.b.Write(v, media)

							case len(contacts.Contacts) != 0:
								b.b.Write(v, contacts)
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
