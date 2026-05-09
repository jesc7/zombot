package planner

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type search struct {
	Until       time.Time
	Sender      string
	Text        string
	LastPID     int64
	Total, M, N int
}

var (
	searches map[string]search
	mu       sync.Mutex
)

func Search(ctx context.Context, db *sql.DB, msg shared.MessageContacts) ([]shared.Contact, error) {
	if searches == nil {
		searches = make(map[string]search)
		go func(ctx context.Context) {
			t1m := time.NewTicker(time.Minute)
			defer t1m.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t1m.C:
					for k, v := range searches {
						if time.Now().After(v.Until) {
							mu.Lock()
							delete(searches, k)
							mu.Unlock()
						}
					}
				}
			}
		}(ctx)
	}

	if msg.Sender == "" || msg.Find == "" {
		return nil, nil
	}

	_new := func(sender, text string) search {
		return search{
			Until:  time.Now().Add(30 * time.Minute),
			Sender: sender,
			Text:   text,
			M:      1,
			N:      8,
		}
	}

	s, ok := searches[msg.Sender]
	if !ok {
		if msg.Find == "/more" {
			return nil, nil
		}
		s = _new(msg.Sender, msg.Find)
		mu.Lock()
		searches[msg.Sender] = s
		mu.Unlock()

		go func(ctx context.Context, key, text string) {
			var cnt int
			if e := db.QueryRowContext(ctx, "select count(1) from pr_getclients_v3(?)", text).Scan(&cnt); e == nil {
				mu.Lock()
				defer mu.Unlock()
				s := searches[key]
				s.Total = cnt
				searches[key] = s
			}
		}(ctx, msg.Sender, msg.Find)

	} else if msg.Find != "/more" {
		s = _new(msg.Sender, msg.Find)
	}

	rows, e := db.QueryContext(ctx, `
		select coalesce(pid, 0), coalesce(cid, 0), caption, phones, address 
		from pr_getclients_v3(?)
		rows ? to ?
	`, s.Text, s.M, s.N)
	if e != nil {
		return nil, e
	}
	defer rows.Close()

	var res []shared.Contact
	for rows.Next() {
		pid, cid, caption, phones, address := int64(0), int64(0), "", "", ""
		if e = rows.Scan(&pid, &cid, &caption, &phones, &address); e != nil {
			continue
		}
		res = append(res, shared.Contact{
			LastPID: types.Iif(len(res) == 0, s.LastPID, 0),
			CID:     cid,
			PID:     pid,
			Caption: caption,
			Phones:  phones,
			Address: address,
		})
		s.M++
	}
	s.N = s.M + 8
	if len(res) != 0 {
		s.LastPID = res[len(res)-1].PID
	}
	mu.Lock()
	searches[msg.Sender] = s
	mu.Unlock()

	return res, nil
}
