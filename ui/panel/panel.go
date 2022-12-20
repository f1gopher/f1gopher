package panel

import (
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
)

type Panel interface {
	Init(dataSrc f1gopherlib.F1GopherLib)

	Draw() (title string, widgets []giu.Widget)

	ProcessTiming(data Messages.Timing)
	ProcessEventTime(data Messages.EventTime)
	ProcessEvent(data Messages.Event)
}
