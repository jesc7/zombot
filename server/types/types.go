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
	WS struct {
		Port   int    `json:"port"`
		JwtKey string `json:"jwt_key"`
	} `json:"ws"`
}
