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

type UniMessageImage struct {
	UniCore
	Files []UniMedia
}

type UniMessageVoice struct {
	UniCore
	Files []UniMedia
}

type UniMessageAudio struct {
	UniCore
	Files []UniMedia
}

type UniMessageVideo struct {
	UniCore
	Files []UniMedia
}

type UniMessageVideoNote struct {
	UniCore
	Files []UniMedia
}

type UniMessageDocument struct {
	UniCore
	Files []UniMedia
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
	Quoted *IUniMessage
	Text   string
}

type UniMessageReaction struct {
	UniCore
	Reacted  *IUniMessage
	Reaction string
}
