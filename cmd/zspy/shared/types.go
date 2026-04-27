package shared

import "time"

/*type MessageType int

const (
	MT_UNDEFINED MessageType = iota - 1
	MT_DUTY
)

type Message struct {
	Type MessageType
}
*/
type MessageText struct {
	Text string `json:"text"`
}

type DutyQuery struct {
	Name string `json:"name"`
	Days int    `json:"days"`
}

type Duty struct {
	Date time.Time `json:"date"`
	Name string    `json:"name"`
}

type MessageDuties struct {
	Q DutyQuery `json:"q"`
	A []Duty    `json:"a,omitempty"`
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
