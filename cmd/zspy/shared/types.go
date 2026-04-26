package shared

import "time"

type MessageDuties struct {
	Q struct {
		Days int    `json:"days"`
		Name string `json:"name"`
	} `json:"q"`
	A struct {
		Duties []struct {
			DutyType int       `json:"duty_type"`
			Date     time.Time `json:"date"`
			Employee string    `json:"Employee"`
		}
	} `json:"a"`
}
