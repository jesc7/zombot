package planner

import (
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

func Search(msg shared.MessageContacts) []shared.Contact {
	_new := func() search {
		return search{
			Until:  time.Now().Add(30 * time.Minute),
			Sender: msg.Sender,
			Text:   msg.Find,
			M:      1,
			N:      8,
		}
	}
	s, ok := searches[msg.Sender]
	if !ok {
		if msg.Find == "/more" {
			return nil
		}
		s = search{
			Until:  time.Now().Add(30 * time.Minute),
			Sender: msg.Sender,
			Text:   msg.Find,
			M:      1,
			N:      8,
		}
		searches[msg.Sender] = s
	} else {
		if msg.Find != "/more" {
			delete()
		}
	}
	return nil
}
