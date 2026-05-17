package planner

import "sync"

type status struct {
	id       int
	caption  string
	name     string
	stateOld string
	stateNew string
}

var (
	statuses    map[int]status
	muClients   sync.Mutex
	onceClients sync.Once
)
