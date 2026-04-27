package shared

import (
	"encoding/json"
	"time"
)

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func Pack[T any](msgType string, data T) (Envelope, error) {
	payloadBytes, e := json.Marshal(data)
	if e != nil {
		return Envelope{}, e
	}
	return Envelope{
		Type:    msgType,
		Payload: payloadBytes,
	}, nil
}

func Unpack[T any](env Envelope) (T, error) {
	var data T
	e := json.Unmarshal(env.Payload, &data)
	return data, e
}

type MessageText struct {
	Text string `json:"text"`
}

type MessageDutyQuery struct {
	Name string `json:"name"`
	Days int    `json:"days"`
}

type Duty struct {
	Date time.Time `json:"date"`
	Name string    `json:"name"`
}

type MessageDuties struct {
	Q MessageDutyQuery `json:"q"`
	A []Duty           `json:"a,omitempty"`
}

type MessageDutyChanges struct {
	A struct {
		Duties []struct {
			Duty
			ChangeType int `json:"change_type"`
		} `json:"duties,omitempty"`
	} `json:"a"`
}

type ZSrvType int

const (
	ZSRV_INFO ZSrvType = iota
	ZSRV_WARN
	ZSRV_PANIC
)

type MessageZSRV struct {
	Status  ZSrvType `json:"status"`
	Caption string   `json:"caption"`
	Text    string   `json:"text"`
}

type MessageCall struct {
	Phone string `json:"phone"`
}
