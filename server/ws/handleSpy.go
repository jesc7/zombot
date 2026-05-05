package ws

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	ctypes "github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/server/types"
)

var BUS_NAMES = []string{types.BUS_BOTMAX, types.BUS_BOTTG}

func (ws *WebSocketServer) handleSpy(ctx context.Context, conn *websocket.Conn, ch chan any) {
	ws.zspy = conn
	defer func() {
		ws.zspy.Close()
		ws.zspy = nil
	}()
	readError := make(chan struct{})

	go func() {
		defer close(readError)

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

		for {
			env, e := shared.Read(ws.zspy)
			if e != nil { //ошибка чтения сокета - выходим
				return
			}

			switch env.Type {
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
				m, e := shared.Unpack[shared.MessageDutyChanges](mt)
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
				m, e := shared.Unpack[shared.MessageAbsents](mt)
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
				m, e := shared.Unpack[shared.MessageBirthdays](mt)
				if e != nil {
					continue
				}
				if len(m.Birthdays) == 0 {
					b.SendText(fmt.Sprintf("☹ В ближайшие %d дней нет ДР", m.Days))
					break
				}

				today := ctypes.ClearTime(time.Now())
				bdToday, bdAfter := []string{}, []string{}
				sb := strings.Builder{}
				for _, v := range m.Birthdays {
					gender := types.RndFrom([2][]string{{"👸🏼", "👸", "👸🏻", "💃"}, {"🤵", "🤵🏻", "🤵🏽"}}[v.Gender]...)
					if v.Date.Equal(today) {
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
				m, e := shared.Unpack[shared.MessageZSRV](mt)
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
				m, e := shared.Unpack[shared.MessageCall](mt)
				if e != nil {
					continue
				}
				text := fmt.Sprintf("📞 Вам звонили%s: <b>%s</b>\n", types.Iif(m.Prefix != "", " на "+m.Prefix, ""), m.Phone)
				if m.Region != "" {
					text += "\n" + m.Region
				}
				b.SendText(text)
			}

			for _, v := range BUS_NAMES {
				ws.b.Write(v, env)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done(): //контекст отменен - выходим
			ws.zspy.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server Shutdown"),
				time.Now().Add(time.Second),
			)
			return

		case <-readError: //ошибка чтения сокета - выходим
			return

		case msg := <-ch: //требуется переслать сообщение клиенту zspy
			switch mt := msg.(type) {
			case shared.Envelope:
				if e := shared.Write(ws.zspy, mt); e != nil {
					log.Println(e)
				}
			}
		}
	}
}
