package shared

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
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

func Read(conn *websocket.Conn) (Envelope, error) {
	mt, data, e := conn.ReadMessage()
	if e != nil {
		return Envelope{}, e
	}

	switch mt {
	case websocket.TextMessage:
		var env Envelope
		e = json.Unmarshal(data, &env)
		return env, e

	default:
		return Envelope{}, nil
	}
}

func Write(conn *websocket.Conn, env Envelope) error {
	data, e := json.Marshal(env)
	if e != nil {
		return e
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

const (
	TypeMessageText        = "message_text"
	TypeMessageDuties      = "message_duties"
	TypeMessageDutyChanges = "message_duty_changes"
	TypeMessageZSRV        = "message_zsrv"
	TypeMessageCall        = "message_call"
	TypeMessageAbsents     = "message_absents"
)

type MessageText struct {
	Text string `json:"text"`
}

type DutyQuery struct {
	Name string `json:"name,omitempty"`
	Days int    `json:"days,omitempty"`
}

type Duty struct {
	Date    time.Time `json:"date"`
	Caption string    `json:"caption"`
}

type MessageDuties struct {
	Q DutyQuery `json:"q,omitempty"`
	A []Duty    `json:"a,omitempty"`
}

type DutyChangeType int

const (
	DCT_NEW DutyChangeType = iota
	DCT_CANCEL
	DCT_REPLACE
)

type MessageDutyChanges struct {
	Changes []struct {
		Duty
		ChangeType DutyChangeType `json:"change_type"`
	} `json:"changes,omitempty"`
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

type EmployeeSex int

const (
	ES_FEMALE EmployeeSex = iota
	ES_MALE
)

type AbsentType int

const (
	AT_DUNNO AbsentType = iota
	AT_ILL
	AT_LEAVE
	AT_DINNER
	AT_OFF
	AT_WORK
)

type MessageAbsents struct {
	Absents []struct {
		Sex  EmployeeSex `json:"sex"`
		Type AbsentType  `json:"type"`
		Name string      `json:"name,omitempty"`
	} `json:"absents"`
}
