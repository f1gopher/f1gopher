package panel

import (
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
)

type Type int

const (
	Info Type = iota
	Timing
	RaceControlMessages
	TrackMap
	Weather
	TeamRadio
	WebTiming
	RacePosition
	GapperPlot
)

func (t Type) String() string {
	return [...]string{"Info", "Timing", "RaceControlMessages", "TrackMap", "Weather", "TeamRadio", "WebTiming", "RacePosition", "GapperPlot"}[t]
}

type Panel interface {
	Type() Type

	Init(dataSrc f1gopherlib.F1GopherLib)
	Close()

	Draw(width int, height int) []giu.Widget

	ProcessTiming(data Messages.Timing)
	ProcessEventTime(data Messages.EventTime)
	ProcessEvent(data Messages.Event)
	ProcessRaceControlMessages(data Messages.RaceControlMessage)
	ProcessWeather(data Messages.Weather)
	ProcessRadio(data Messages.Radio)
	ProcessLocation(data Messages.Location)
}
