package types

type Config struct {
	Max struct {
		Token string `json:"token"`
	} `json:"max"`
	Proxy struct {
		Addr string `json:"addr"`
		Port int    `json:"port"`
	} `json:"proxy"`
}
