package planner

import "time"

type MessengerType int

const (
	MT_MAX MessengerType = iota
	MT_TELEGRAM
)

type Search struct {
	Until       time.Time
	MT          MessengerType
	Sender      string
	Text        string
	Total, M, N int
}

var searches map[string]Search
