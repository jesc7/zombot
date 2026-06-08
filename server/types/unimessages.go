package types

/*
	UNIVERSAL MESSAGES
	way to send a messages in any supported messengers
*/

type IUniMessage interface {
	UniID() string
}

type UniCore struct {
	ID      string
	Caption string
	Text    string
}

func (m UniCore) UniID() string {
	return m.ID
}

type UniMessageText struct {
	UniCore
}

type IUniMedia interface {
	UniData() []byte
}

type UniMedia struct {
	Name string
	Data []byte
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

func (msg UniMessageMedia) IsCollage() bool {
	var count int
	for _, m := range msg.Media {
		switch m.(type) {
		case UniImage, UniVideo, UniVideoNote:
			count++
		}
	}
	return count == len(msg.Media)
}

func (msg UniMessageMedia) IsPhotoCollage() bool {
	if len(msg.Media) == 0 {
		return false
	}
	_, ok := msg.Media[0].(UniImage)
	return ok
}

func (msg UniMessageMedia) IsVideoCollage() bool {
	if len(msg.Media) == 0 {
		return false
	}
	switch msg.Media[0].(type) {
	case UniVideo, UniVideoNote:
		return true
	default:
		return false
	}
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
}

type UniMessageReaction struct {
	UniCore
	Reacted  *IUniMessage
	Reaction string
}
