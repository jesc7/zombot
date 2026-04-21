package bot

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	max "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	_ "github.com/nakagami/firebirdsql"
	"golang.org/x/time/rate"

	"github.com/jesc7/zombot/jp/duties"
	"github.com/jesc7/zombot/queue"
	"github.com/jesc7/zombot/types"
)

type TextMsg struct {
	Text string
}
type Bot struct {
	bot    *max.Api
	QWait  *queue.Queue
	chatID int64
	db     *sql.DB
}

func NewBot(ctx context.Context, cfg types.Config) (*Bot, error) {
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

	db, e := sql.Open(cfg.DB.Driver, cfg.DB.ConnStr)
	if e != nil {
		return nil, e
	}

	bot, e := max.New(cfg.Max.Token, options...)
	return &Bot{
		bot:    bot,
		QWait:  queue.NewQ(ctx, rate.Limit(5)),
		db:     db,
		chatID: cfg.Max.ChatID,
	}, e
}

func (b *Bot) Free() {
	if b.db != nil {
		b.db.Close()
	}
}

func (b *Bot) SendText(text string) {
	b.QWait.Add(&queue.WaitObj{
		O: max.NewMessage().
			SetText(text).
			SetFormat(schemes.HTML),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) SendCall(phone string) {
	b.QWait.Add(&queue.WaitObj{
		O: max.NewMessage().
			SetText(fmt.Sprintf("📞 Вам звонили%s: <b>%s</b>\n", types.Iif(strings.HasPrefix(phone, "8800 "), " на 8800", ""), phone)).
			SetFormat(schemes.HTML),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) SendZSrv(msg types.ZSrvMessage) {
	if strings.Count(msg.Text, "\n") != 0 {
		msg.Text = "\n" + msg.Text
	}
	switch msg.Status {
	case types.ZMSG_WARN:
		msg.Text = fmt.Sprintf("⚠️ <i>zsrv %s беспокоится</i>\n%s", msg.Caption, msg.Text)
	case types.ZMSG_PANIC:
		msg.Text = fmt.Sprintf("🆘 <i>zsrv %s паникует</i>\n%s", msg.Caption, msg.Text)
	default:
		msg.Text = fmt.Sprintf("ℹ <i>zsrv %s информирует</i>\n%s", msg.Caption, msg.Text)
	}
	b.QWait.Add(&queue.WaitObj{
		O: max.NewMessage().
			SetText(msg.Text).
			SetFormat(schemes.HTML),
	}, queue.PRIORITY_NORMAL)
}

func (b *Bot) Run(ctx context.Context) {
out:
	for {
		select {
		case <-ctx.Done():
			break out

		case m := <-b.QWait.Q: //разгребаем локальную очередь сообщений
			wo, ok := m.(*queue.WaitObj)
			if !ok {
				break
			}
			msg, ok := wo.O.(*max.Message)
			if !ok {
				break
			}
			_ = msg
			if wo.OnOk != nil {
				wo.OnOk()
			}

		case update := <-b.bot.GetUpdates(ctx): //приехали апдейты с сервера
			switch upd := update.(type) {
			case *schemes.MessageCreatedUpdate:
				//только групповой чат из настроек
				if upd.Message.Recipient.ChatType != schemes.CHAT || upd.GetChatID() != b.chatID {
					break
				}

				switch upd.GetCommand() {
				case "/duty": //дежурства
					text := upd.GetParam()
					dut, _ := duties.DutiesList(b.db)
					if i, e := strconv.Atoi(text); e == nil && i > 0 && i < 365 {
						text = duties.Duties(b.db, i, dut, "")
					} else {
						text = duties.Duties(b.db, 7, dut, text)
					}
					b.QWait.Add(&queue.WaitObj{
						O: max.NewMessage().
							SetFormat(schemes.HTML).
							SetChat(upd.GetChatID()).
							SetText(text),
					}, queue.PRIORITY_NORMAL)

				case "/absent":
				case "/birthday":
				case "/ratings":
				case "/ci":

				case "/chatid":
					if e := b.bot.Messages.Send(ctx, max.NewMessage().
						SetFormat(schemes.HTML).
						SetChat(upd.GetChatID()).
						SetText("ChatID: "+strconv.FormatInt(upd.GetChatID(), 64))); e != nil {
						log.Println("Send message error:", e)
					}

				default:
				}
			}
		}
	}
}
