package bot

import (
	tg "github.com/mymmrac/telego"
	//th "github.com/mymmrac/telego/telegohandler"
	//tu "github.com/mymmrac/telego/telegoutil"

	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/cmd/zspy/shared/bus"
	"github.com/jesc7/zombot/server/queue"
)

type Bot struct {
	*tg.Bot
	QWait  *queue.Queue
	chatID int64
	b      *bus.Bus
	ch     chan shared.Envelope
}
