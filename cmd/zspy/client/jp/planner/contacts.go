package planner

import (
	"time"

	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type search struct {
	Until       time.Time
	MT          shared.MessengerType
	Sender      string
	Text        string
	Total, M, N int
}

var searches = make(map[string]search)

func Search(msg shared.MessageContacts) []shared.Contact {
	s, ok := searches[msg.Sender]
	if !ok && msg.Find == "/more" {
		return nil
	}
	return nil
}
