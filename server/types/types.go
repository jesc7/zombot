package types

type Config struct {
	Max struct {
		Token  string `json:"token"`
		ChatID int64  `json:"chat_id"`
	} `json:"max"`
	TG struct {
		Token  string `json:"token"`
		ChatID int64  `json:"chat_id"`
	} `json:"tg"`
	Proxy struct {
		Addr string `json:"addr"`
		Port int    `json:"port"`
	} `json:"proxy"`
	WS struct {
		Port   int    `json:"port"`
		JwtKey string `json:"jwt_key"`
	} `json:"ws"`
}

const (
	BUS_BOTMAX = "bot_max"
	BUS_BOTTG  = "bot_tg"
	BUS_WS     = "ws"
)

type UniMessageText struct {
	Text string
}
type UniMessageCaption struct {
	Caption string
}
