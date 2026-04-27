package shared

import "time"

type MessageType int

const (
	MT_UNDEFINED MessageType = iota - 1
	MT_DUTY
)

type Message struct {
	Type MessageType
}

type MessageText struct {
	Text string `json:"text"`
}

type DutyQuery struct {
	Name string `json:"name"`
	Days int    `json:"days"`
}

type DutyAnswer struct {
	Date time.Time `json:"date"`
	Name string    `json:"name"`
}

type MessageDuties struct {
	Q DutyQuery    `json:"q"`
	A []DutyAnswer `json:"a,omitempty"`
}

type MessageDutyChanges struct {
	A struct {
		Duties []struct {
			DutyAnswer
			ChangeType int `json:"change_type"`
		} `json:"duties,omitempty"`
	} `json:"a"`
}
