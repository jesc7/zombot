package planner

import "sync"

type status struct {
	agentID  int64
	id       int64
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
