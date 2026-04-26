package shared

import "time"

type MessageDuties struct {
	Q struct {
		Days int    `json:"days"`
		Name string `json:"name"`
	} `json:"q"`
	A struct {
		Duties []struct {
			Date     time.Time `json:"date"`
			Employee string    `json:"employee"`
		} `json:"duties,omitempty"`
	} `json:"a"`
}

type MessageDutyChanges struct {
	A struct {
		Duties []struct {
			ChangeType int       `json:"change_type"`
			Date       time.Time `json:"date"`
			Employee   string    `json:"employee"`
		} `json:"duties,omitempty"`
	} `json:"a"`
}
