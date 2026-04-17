package types

type Config struct {
	Max struct {
		Token  string `json:"token"`
		ChatID int64  `json:"chat_id"`
	} `json:"max"`
	Proxy struct {
		Addr string `json:"addr"`
		Port int    `json:"port"`
	} `json:"proxy"`
}

type MsgType int

const (
	ZMSG_INFO MsgType = iota
	ZMSG_WARN
	ZMSG_PANIC
)

type ZSrvMessage struct {
	Status  MsgType `json:"status"`
	Caption string  `json:"caption"`
	Text    string  `json:"text"`
}
