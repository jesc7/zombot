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

func (m *UniCore) UniID() string {
	return m.ID
}

type UniMessageText struct {
	UniCore
	Text string
}

type UniMessageFile struct {
	UniCore
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
	UniCore
	Contact
}

type UniMessageContacts struct {
	UniCore
	Contacts []Contact
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
