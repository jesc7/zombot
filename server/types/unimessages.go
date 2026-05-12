package types

/*
	UNIVERSAL MESSAGES
	way to send a messages in any supported messengers
*/

type UniMessage interface {
	UniID() string
}

type UniCore struct {
	ID string
}

func (m UniCore) UniID() string {
	return m.ID
}

type UniMessageText struct {
	UniCore
	Text string
}

type UniFile struct {
	Name    string
	Caption string
	File    []byte
}

type UniMessageImage struct {
	UniCore
	Files []UniFile
}

type UniMessageVoice struct {
	UniCore
	Files []UniFile
}

type UniMessageAudio struct {
	UniCore
	Files []UniFile
}

type UniMessageVideo struct {
	UniCore
	Files []UniFile
}

type UniMessageVideoNote struct {
	UniCore
	Files []UniFile
}

type UniMessageDocument struct {
	UniCore
	Files []UniFile
}

type UniContact struct {
}

type UniMessageContact struct {
	UniCore
	UniContact
}

type UniMessageContacts struct {
	UniCore
	Contacts []UniContact
}

type UniMessageQuoted struct {
	UniCore
	Quoted *UniMessage
	Text   string
}

type UniMessageReaction struct {
	UniCore
	Reacted  *UniMessage
	Reaction string
}
