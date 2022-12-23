package panel

import (
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
)

type Panel interface {
	Init(dataSrc f1gopherlib.F1GopherLib)
	Close()

	Draw() (title string, widgets []giu.Widget)

	ProcessTiming(data Messages.Timing)
	ProcessEventTime(data Messages.EventTime)
	ProcessEvent(data Messages.Event)
	ProcessRaceControlMessages(data Messages.RaceControlMessage)
	ProcessWeather(data Messages.Weather)
	ProcessRadio(data Messages.Radio)
	ProcessLocation(data Messages.Location)
}
