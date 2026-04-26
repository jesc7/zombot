package shared

import "time"

type MessageDuties struct {
	Days   int    `json:"days"`
	Name   string `json:"name"`
	Duties []struct {
		Date     time.Time `json:"date"`
		Employee string    `json:"Employee"`
	}
}
