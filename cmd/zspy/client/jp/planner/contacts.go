package planner

import (
	"context"
	"database/sql"
	"time"

	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type search struct {
	Until       time.Time
	Sender      string
	Text        string
	LastPID     int64
	Total, M, N int
}

var searches map[string]search

func Search(ctx context.Context, db *sql.DB, msg shared.MessageContacts) ([]shared.Contact, error) {
	if searches == nil {
		searches = make(map[string]search)
		go func() {
			t1m := time.NewTicker(time.Minute)
			defer t1m.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				}
			}
		}()
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
		searches[msg.Sender] = s
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
			CID:     cid,
			PID:     pid,
			Caption: caption,
			Phones:  phones,
			Address: address,
		})
		s.M++
	}
	s.N = s.M + 8
	searches[msg.Sender] = s

	return res, nil
}
