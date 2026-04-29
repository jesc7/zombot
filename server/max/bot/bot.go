package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	max "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"golang.org/x/time/rate"

	ctypes "github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	"github.com/jesc7/zombot/server/queue"
	"github.com/jesc7/zombot/server/types"
)

const (
	MSG_HELP = `<b>Команды бота:</b>

дежур[...] [кто] [дней] - дежурства
отсутств[...] - кто отсутствует
`
)

type Bot struct {
	bot    *max.Api
	QWait  *queue.Queue
	chatID int64
	b      *bus.Bus
	ch     chan shared.Envelope
}

func NewBot(ctx context.Context, cfg types.Config, b *bus.Bus) (*Bot, error) {
	ch, e := b.Register(types.BUS_BOT)
	if e != nil {
		return nil, e
	}

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
		ch:     ch,
	}, e
}

func (b *Bot) SendText(text string) {
	b.QWait.Add(&queue.WaitObj{
		O: max.NewMessage().
			SetText(text),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) Run(ctx context.Context) {

	_tipDay := func(t time.Time) string {
		today, tip := ctypes.ClearTime(time.Now()), ""
		switch t {
		case today:
			tip = " (сегодня)"
		case today.Add(24 * time.Hour):
			tip = " (завтра)"
		case today.Add(24 * time.Hour * 2):
			tip = " (послезавтра)"
		}
		return tip
	}

out:
	for {
		select {
		case <-ctx.Done():
			break out

		case env := <-b.ch: //разгребаем пакеты, пришедшие боту
			log.Println("Bot", env.Type)

			switch env.Type {
			//просто текст
			case shared.TypeMessageText:
				m, e := shared.Unpack[shared.MessageText](env)
				if e != nil {
					continue
				}
				b.SendText(m.Text)

			//дежурства
			case shared.TypeMessageDuties:
				m, e := shared.Unpack[shared.MessageDuties](env)
				if e != nil {
					continue
				}
				if len(m.A) == 0 {
					b.SendText("😟 Дежурства не найдены")
					break
				}

				sb := strings.Builder{}
				sb.WriteString("👷 <b>Дежурные</b>\n\n")
				for _, v := range m.A {
					fmt.Fprintf(&sb, "%s%s: %s\n", v.Date.Format("02.01"), _tipDay(v.Date), v.Caption)
				}
				b.SendText(sb.String())

			//изменения дежурств
			case shared.TypeMessageDutyChanges:
				m, e := shared.Unpack[shared.MessageDutyChanges](env)
				if e != nil || len(m.Changes) == 0 {
					continue
				}

				sb := strings.Builder{}
				sb.WriteString("👷 <b>Изменения дежурств</b>\n\n")
				signs := []string{"⭐", "🚫", "🔄"}
				for _, v := range m.Changes {
					fmt.Fprintf(&sb, "%s %s%s: %s\n", signs[v.ChangeType], v.Date.Format("02.01"), _tipDay(v.Date), v.Caption)
				}
				b.SendText(sb.String())

			//отсутствующие
			case shared.TypeMessageAbsents:
				m, e := shared.Unpack[shared.MessageAbsents](env)
				if e != nil {
					continue
				}
				if len(m.Absents) == 0 {
					b.SendText("🙂 Все на месте")
					break
				}

				sb := strings.Builder{}
				sb.WriteString("👤 <b>Отсутствующие</b>\n\n")
				for _, v := range m.Absents {
					var tip string
					switch v.Type {
					case shared.AT_DUNNO:
						tip = types.Dunno(int(v.Gender)) //неизвестно
					case shared.AT_ILL:
						tip = types.RndFrom("🤕", "😷", "🤧", "🤒") //больничный
					case shared.AT_LEAVE:
						tip = types.RndFrom("🏖", "⛱️", "🏕️", "🏝️", "⛰️", "✈️") //отпуск
					case shared.AT_DINNER:
						tip = types.RndFrom("🍔", "🍳", "🥘", "🥗", "🍱") //обед
					case shared.AT_OFF:
						tip = types.RndFrom([2][]string{{"🚶‍♀️", "🏃‍♀️"}, {"🚶🏻‍♂️", "🏃‍♂️"}}[v.Gender]...) //ушел
					case shared.AT_WORK:
						tip = types.RndFrom([2][]string{{"👷‍♀️", "👩‍🔧"}, {"👷", "👨‍🔧"}}[v.Gender]...) //по рабочим делам
					default:
						tip = types.Dunno(int(v.Gender)) //неизвестно
					}
					fmt.Fprintf(&sb, "%s %s%s\n", tip, v.Name, types.Iif(len(v.Comment) != 0, " - "+v.Comment, ""))
				}
				b.SendText(sb.String())

			//дни рождения
			case shared.TypeMessageBirthdays:
				m, e := shared.Unpack[shared.MessageBirthdays](env)
				if e != nil {
					continue
				}
				if len(m.Birthdays) == 0 {
					b.SendText("☹ В ближайший месяц нет дней рождения")
					break
				}

				today := ctypes.ClearTime(time.Now())
				bdToday, bdAfter := []string{}, []string{}
				sb := strings.Builder{}
				for _, v := range m.Birthdays {
					gender := types.RndFrom([2][]string{{"👸🏼", "👸", "👸🏻", "💃"}, {"🤵", "🤵🏻", "🤵🏽"}}[v.Gender]...)
					if v.Date == today {
						bdToday = append(bdToday, fmt.Sprintf("%s %s", gender, v.Caption))
					} else {
						bdAfter = append(bdAfter, fmt.Sprintf("%s %s (%s)", gender, v.Caption, v.Date.Format("02.01")))
					}
				}
				if len(bdToday) != 0 {
					tip := []string{"🎉", "🎁", "🎂", "✨", "💐"}
					fmt.Fprintf(&sb, "<b>Сегодня день рождения у:</b>\n%s\n\nПоздравляем, ю-ху!!! %s%s%s",
						strings.Join(bdToday, "\n"),
						types.RndFrom(tip...),
						types.RndFrom(tip...),
						types.RndFrom(tip...))
					if len(bdAfter) != 0 {
						sb.WriteString("\n\n<b>А еще скоро день рождения у:</b>\n")
					}
				} else if len(bdAfter) != 0 {
					sb.WriteString("<b>Скоро день рождения у:</b>\n\n")
				}
				sb.WriteString(strings.Join(bdAfter, "\n"))
				b.SendText(sb.String())

			//сообщения от площадок
			case shared.TypeMessageZSRV:
				m, e := shared.Unpack[shared.MessageZSRV](env)
				if e != nil {
					continue
				}
				if strings.Count(m.Text, "\n") != 0 {
					m.Text = "\n" + m.Text
				}
				switch m.Status {
				case shared.ZSRV_WARN:
					m.Text = fmt.Sprintf("⚠️ <i>zsrv %s беспокоится</i>\n%s", m.Caption, m.Text)
				case shared.ZSRV_PANIC:
					m.Text = fmt.Sprintf("🆘 <i>zsrv %s паникует</i>\n%s", m.Caption, m.Text)
				default:
					m.Text = fmt.Sprintf("ℹ <i>zsrv %s информирует</i>\n%s", m.Caption, m.Text)
				}
				b.SendText(m.Text)

			//звонки
			case shared.TypeMessageCall:
				m, e := shared.Unpack[shared.MessageCall](env)
				if e != nil {
					continue
				}
				b.SendText(fmt.Sprintf("📞 Вам звонили%s: <b>%s</b>\n", types.Iif(strings.HasPrefix(m.Phone, "8800 "), " на 8800", ""), m.Phone))
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
				log.Println("Message from", upd.GetChatID())

				//только групповой чат из настроек
				if upd.Message.Recipient.ChatType != schemes.CHAT || upd.GetChatID() != b.chatID {
					break
				}

				if isHelp(upd.Message.Body.Text) {
					upd.Message.Body.Text = "/help"
				} else if duty, name, days := isDuty(upd.Message.Body.Text); duty {
					upd.Message.Body.Text = fmt.Sprintf("/duty:%s#%d", name, days)
				} else if isAbsent(upd.Message.Body.Text) {
					upd.Message.Body.Text = "/absent"
				} else if bd, days := isBirthday(upd.Message.Body.Text); bd {
					upd.Message.Body.Text = fmt.Sprintf("/birthday:%d", days)
				}

				switch upd.GetCommand() {
				case "/help": //помощь
					b.SendText(MSG_HELP)

				case "/duty": //дежурства
					params := strings.Split(upd.GetParam(), "#")
					name, days := params[0], 7
					if len(params) > 1 {
						days, _ = strconv.Atoi(params[1])
					}
					env, e := shared.Pack(shared.TypeMessageDuties, shared.MessageDuties{
						Q: shared.DutyQuery{
							Name: name,
							Days: days,
						},
					})
					if e != nil {
						break
					}
					b.b.Write(types.BUS_WS, env)

				case "/absent": //отсутствующие
					env, e := shared.Pack(shared.TypeMessageAbsents, shared.MessageAbsents{})
					if e != nil {
						break
					}
					b.b.Write(types.BUS_WS, env)

				case "/birthday": //дни рождения
					days, _ := strconv.Atoi(upd.GetParam())
					if days <= 0 {
						days = 31
					}
					env, e := shared.Pack(shared.TypeMessageBirthdays, shared.MessageBirthdays{Days: days})
					if e != nil {
						break
					}
					b.b.Write(types.BUS_WS, env)

				case "/ratings": //пятничные рейтинги
				case "/ci": //инфо о клиентах
				}
			}
		}
	}
}
