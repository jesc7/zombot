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
	Total, M, N int
}

var searches = make(map[string]search)

func Search(ctx context.Context, db *sql.DB, msg shared.MessageContacts) []shared.Contact {
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
			return nil
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
		return nil
	}

	return nil
}
