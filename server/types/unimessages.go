package types

/*
	UNIVERSAL MESSAGES
	way to send a messages in any supported messengers
*/

type IUniMessage interface {
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

type IUniMedia interface {
	UniData() []byte
}

type UniMedia struct {
	Name    string
	Caption string
	Data    []byte
}

func (m UniMedia) UniData() []byte {
	return m.Data
}

type UniImage struct {
	UniMedia
	Ext string
}

type UniVoice struct {
	UniMedia
}

type UniAudio struct {
	UniMedia
}

type UniVideo struct {
	UniMedia
}

type UniVideoNote struct {
	UniMedia
}

type UniDocument struct {
	UniMedia
}

type UniMessageMedia struct {
	UniCore
	Media []IUniMedia
}

type Contact struct {
	Caption string
	Phone   string
}

type UniMessageContacts struct {
	UniCore
	Contacts []Contact
}

type UniMessageQuoted struct {
	UniCore
	Quoted *IUniMessage
	Text   string
}

type UniMessageReaction struct {
	UniCore
	Reacted  *IUniMessage
	Reaction string
}
