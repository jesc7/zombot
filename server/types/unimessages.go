package types

/*
	UNIVERSAL MESSAGES
	way to send a messages in any supported messengers
*/

type UniMessage interface {
	UniID() string
}

type UniMessageCore struct {
	ID string
}

func (m *UniMessageCore) UniID() string {
	return m.ID
}

type UniMessageText struct {
	UniMessageCore
	Text string
}

type UniMessageFile struct {
	UniMessageCore
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
	UniMessageCore
	Contact
}

type UniMessageContacts struct {
	UniMessageCore
	Contacts []Contact
}

type UniMessageQuoted struct {
	UniMessageCore
	Quoted *UniMessage
	Text   string
}

type UniMessageReaction struct {
	UniMessageCore
	Reacted  *UniMessage
	Reaction string
}
