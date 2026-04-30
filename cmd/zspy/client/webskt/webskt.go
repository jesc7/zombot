package webskt

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/nakagami/firebirdsql"

	"github.com/jesc7/zombot/cmd/zspy/client/jp/checks"
	"github.com/jesc7/zombot/cmd/zspy/client/jp/duties"
	"github.com/jesc7/zombot/cmd/zspy/client/jp/planner"
	"github.com/jesc7/zombot/cmd/zspy/client/phones"
	"github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type WebSocketClient struct {
	host   url.URL
	header http.Header
	ch     chan shared.Envelope
	conn   *websocket.Conn
	cwd    string
}

func NewWebSocketClient(cfg types.Config, cwd string) *WebSocketClient {
	return &WebSocketClient{
		host:   url.URL{Scheme: "ws", Host: cfg.Host, Path: "/ws"},
		header: http.Header{"Authorization": []string{"Bearer " + cfg.Token}},
		cwd:    cwd,
	}
}

func (ws *WebSocketClient) Write(env shared.Envelope) {
	defer recover()
	ws.ch <- env
}

func (ws *WebSocketClient) Run(ctx context.Context, cfg types.Config) error {
	db, e := sql.Open(cfg.DB.Driver, cfg.DB.ConnStr)
	if e != nil {
		return e
	}
	defer db.Close()

	ws.ch = make(chan shared.Envelope)
	defer close(ws.ch)

	for {
		if ws.conn, _, e = websocket.DefaultDialer.DialContext(ctx, ws.host.String(), ws.header); e != nil {
			log.Println(e)
			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-time.After(10 * time.Second):
				continue
			}
		}
		ws.handle(ctx, cfg, db)
	}
}

func (ws *WebSocketClient) handle(ctx context.Context, cfg types.Config, db *sql.DB) {

	log.Printf("Connected: %s / %s", ws.conn.LocalAddr(), ws.conn.RemoteAddr())

	defer ws.conn.Close()
	readError := make(chan struct{})

	go func() {
		defer close(readError)

		for {
			env, e := shared.Read(ws.conn)
			if e != nil {
				return
			}

			switch env.Type {
			case shared.TypeMessageDuties:
				pay, e := shared.Unpack[shared.MessageDuties](env)
				if e != nil {
					continue
				}
				pay.A, e = duties.Duty(ctx, db, pay.Q)
				if e != nil {
					continue
				}
				env, e = shared.Pack(env.Type, pay)
				if e != nil {
					continue
				}
				ws.Write(env)

			case shared.TypeMessageAbsents:
				pay, e := planner.Absents(ctx, db)
				if e != nil {
					continue
				}
				env, e = shared.Pack(env.Type, shared.MessageAbsents{
					Absents: pay,
				})
				if e != nil {
					continue
				}
				ws.Write(env)

			case shared.TypeMessageBirthdays:
				pay, e := shared.Unpack[shared.MessageBirthdays](env)
				if e != nil {
					continue
				}
				pay.Birthdays, e = planner.Birthdays(ctx, db, pay.Days)
				if e != nil {
					continue
				}
				env, e = shared.Pack(env.Type, pay)
				if e != nil {
					continue
				}
				ws.Write(env)
			}
		}
	}()

	tPing := time.NewTicker(10 * time.Second)
	defer tPing.Stop()
	t1m := time.NewTicker(1 * time.Minute)
	defer t1m.Stop()
	t5m := time.NewTicker(5 * time.Minute)
	defer t5m.Stop()
	t30m := time.NewTicker(30 * time.Minute)
	defer t30m.Stop()

	t08_00 := time.NewTicker(types.NextTime("08:00"))
	defer t08_00.Stop()
	t08_10 := time.NewTicker(types.NextTime("08:10"))
	defer t08_10.Stop()
	t09_00 := time.NewTicker(types.NextTime("09:00"))
	defer t09_00.Stop()
	t18_00 := time.NewTicker(types.NextTime("18:00"))
	defer t18_00.Stop()
	t20_00 := time.NewTicker(types.NextTime("20:00"))
	defer t20_00.Stop()

	for {
		select {
		case <-ctx.Done(): //контекст отменен - выходим
			ws.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return

		case <-readError: //ошибка чтения сокета - выходим
			return

		case <-tPing.C: //ошибка отправки ping - выходим
			if e := ws.conn.WriteMessage(websocket.PingMessage, nil); e != nil {
				return
			}

		case <-t08_00.C: //everyday 8:00
			t08_00.Reset(24 * time.Hour)

			go func() { //update phone base
				phones.PbUpdate(ws.cwd, []string{})
			}()

			go func() { //checks EC
				if s := checks.CheckEC(cfg.EC); s != "" {
					if env, e := shared.Pack(shared.TypeMessageText, shared.MessageText{
						Text: s,
					}); e == nil {
						ws.Write(env)
					}
				}
			}()

		case <-t08_10.C: //everyday 8:10
			t08_10.Reset(24 * time.Hour)

			go func() { //birthdays today
				var (
					pay shared.MessageBirthdays
					e   error
				)
				pay.Birthdays, e = planner.Birthdays(ctx, db, 1)
				if e != nil || len(pay.Birthdays) == 0 {
					return
				}
				env, e := shared.Pack(shared.TypeMessageBirthdays, pay)
				if e != nil {
					return
				}
				ws.Write(env)
			}()

			go func() { //another countries holiday
				if s := planner.ForeignHoliday(); s != "" {
					if env, e := shared.Pack(shared.TypeMessageText, shared.MessageText{
						Text: s,
					}); e == nil {
						ws.Write(env)
					}
				}
			}()

			go func() { //check domains registration
				if s := checks.CheckWhois(cfg.CheckDomains, 10); s != "" {
					if env, e := shared.Pack(shared.TypeMessageText, shared.MessageText{
						Text: s,
					}); e == nil {
						ws.Write(env)
					}
				}
			}()

		case <-t09_00.C: //everyday 9:00
			t09_00.Reset(24 * time.Hour)

			go func() { //who's absent today
				pay, e := planner.Absents(ctx, db)
				if e != nil || len(pay) == 0 {
					return
				}
				env, e := shared.Pack(shared.TypeMessageAbsents, shared.MessageAbsents{
					Absents: pay,
				})
				if e != nil {
					return
				}
				ws.Write(env)
			}()

			go func() { //missing duties
				if s := duties.MissDuties(ctx, db, 20); s != "" {
					if env, e := shared.Pack(shared.TypeMessageText, shared.MessageText{
						Text: s,
					}); e == nil {
						ws.Write(env)
					}
				}
			}()

		case <-t18_00.C: //everyday 18:00
			t18_00.Reset(24 * time.Hour)

			go func() { //tomorrow duties
				if s := duties.TomorrowDuties(ctx, db); s != "" {
					if env, e := shared.Pack(shared.TypeMessageText, shared.MessageText{
						Text: s,
					}); e == nil {
						ws.Write(env)
					}
				}
			}()

		case <-t20_00.C: //everyday 20:00
			t20_00.Reset(24 * time.Hour)

			go func() { //tomorrow duties
				if s := duties.TomorrowDuties(ctx, db); s != "" {
					if env, e := shared.Pack(shared.TypeMessageText, shared.MessageText{
						Text: s,
					}); e == nil {
						ws.Write(env)
					}
				}
			}()

			go func() { //find duties for next 2 days
				var (
					pay shared.MessageDuties
					e   error
				)
				pay.Q.Days = 2
				pay.A, e = duties.Duty(ctx, db, pay.Q)
				if e != nil || len(pay.A) == 0 {
					return
				}
				env, e := shared.Pack(shared.TypeMessageDuties, pay)
				if e != nil {
					return
				}
				ws.Write(env)
			}()

		case <-t1m.C: //every 1 minutes
			go func() { //End-of-work list
				if s := planner.EowList(ctx, db); s != "" {
					if env, e := shared.Pack(shared.TypeMessageText, shared.MessageText{
						Text: s,
					}); e == nil {
						ws.Write(env)
					}
				}
			}()

		case <-t5m.C: //every 5 minutes
			go func() { //zsrv watcher
				if s := checks.WatchZsrv(cfg.ZSrv); s != "" {
					if env, e := shared.Pack(shared.TypeMessageText, shared.MessageText{
						Text: s,
					}); e == nil {
						ws.Write(env)
					}
				}
			}()

		case <-t30m.C: //every 30 minutes
			go func() { //critical tasks
				if s := planner.WatchCriticalTasks(ctx, db, 30); s != "" {
					if env, e := shared.Pack(shared.TypeMessageText, shared.MessageText{
						Text: s,
					}); e == nil {
						ws.Write(env)
					}
				}
			}()

		case env := <-ws.ch: //наконец-то делаем что-то полезное
			if e := shared.Write(ws.conn, env); e != nil { //send message to WebSocket server
				log.Println(e)
			}
		}
	}
}
