package types

/*
	UNIVERSAL MESSAGES
	way to send a messages in any supported messengers
*/

type IntUniMessage interface {
	UniID() string
}

type UniMessage struct {
	ID string
}

func (m *UniMessage) UniID() string {
	return m.ID
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
	Quoted IntUniMessage
	Text   string
}

type UniMessageReaction struct {
	UniMessage
	Reacted  IntUniMessage
	Reaction string
}
