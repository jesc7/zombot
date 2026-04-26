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
		Port int    `json:"port"`
		JWT  string `json:"jwt"`
	} `json:"ws"`
	DB struct {
		Driver  string `json:"driver"`
		ConnStr string `json:"connstr"`
	} `json:"db"`
	CheckDomains []string    `json:"check_domains"`
	Checks       []string    `json:"checks"`
	CFChecks     []string    `json:"cf_checks"`
	ZSrv         []ZSrvWatch `json:"zsrv"`
	EC           EC          `json:"ec"`
}

type ZSrvWatch struct {
	Url     string `json:"url"`
	Caption string `json:"caption"`
}

type EC struct {
	Driver  string `json:"driver"`
	ConnStr string `json:"connstr"`
	Pwd     string `json:"pwd"`
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
