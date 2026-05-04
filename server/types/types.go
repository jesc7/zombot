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

/*
	UNIVERSAL MESSAGES
	way to send a messages in any supported messengers
*/

type UniMessage struct {
	ID string
}

type UniMessageText struct {
	UniMessage
	Text string
}

type UniMessageFile struct {
	UniMessage
	Name    string
	Caption string
	File    []byte
}

type UniMessageImage struct {
	UniMessageFile
}

type UniMessageVoice struct {
	UniMessageFile
}

type UniMessageAudio struct {
	UniMessageFile
}

type UniMessageVideo struct {
	UniMessageFile
}

type UniMessageVideoNote struct {
	UniMessageFile
}

type UniMessageDocument struct {
	UniMessageFile
}

type Contact struct {
}

type UniMessageContact struct {
	UniMessage
	Contact
}

type UniMessageContacts struct {
	UniMessage
	Contacts []Contact
}

type UniMessageQuoted struct {
	UniMessage
	Quoted UniMessage
	Text   string
}

type UniMessageReaction struct {
	UniMessage
	Reacted  UniMessage
	Reaction string
}
