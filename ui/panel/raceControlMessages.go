package panel

import (
	"fmt"
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"golang.org/x/image/colornames"
	"strings"
	"sync"
	"sync/atomic"
)

type raceControlMessages struct {
	dataSrc        f1gopherlib.F1GopherLib
	rcMessages     []Messages.RaceControlMessage
	rcMessagesLock sync.Mutex
	dataChanged    atomic.Bool

	cachedUI []giu.Widget
}

func CreateRaceControlMessages() Panel {
	return &raceControlMessages{
		rcMessages: make([]Messages.RaceControlMessage, 0),
	}
}

func (r *raceControlMessages) ProcessTiming(data Messages.Timing)       {}
func (r *raceControlMessages) ProcessEventTime(data Messages.EventTime) {}
func (r *raceControlMessages) ProcessEvent(data Messages.Event)         {}
func (r *raceControlMessages) ProcessWeather(data Messages.Weather)     {}
func (r *raceControlMessages) ProcessRadio(data Messages.Radio)         {}
func (r *raceControlMessages) ProcessLocation(data Messages.Location)   {}
func (r *raceControlMessages) Close()                                   {}

func (r *raceControlMessages) Type() Type { return RaceControlMessages }

func (r *raceControlMessages) Init(dataSrc f1gopherlib.F1GopherLib) {
	r.dataSrc = dataSrc

	// Clear previous session data
	r.rcMessages = make([]Messages.RaceControlMessage, 0)
	r.cachedUI = make([]giu.Widget, 0)
}

func (r *raceControlMessages) ProcessRaceControlMessages(data Messages.RaceControlMessage) {
	r.rcMessagesLock.Lock()
	r.rcMessages = append(r.rcMessages, data)
	r.rcMessagesLock.Unlock()
	r.dataChanged.Store(true)
}

func (r *raceControlMessages) Draw(width int, height int) []giu.Widget {

	if r.dataChanged.CompareAndSwap(true, false) {
		r.dataChanged.Store(false)
		r.cachedUI = r.formatMessages()
	}

	return r.cachedUI
}

func (r *raceControlMessages) formatMessages() []giu.Widget {
	msgs := make([]giu.Widget, 0)

	r.rcMessagesLock.Lock()
	if len(r.rcMessages) > 0 {
		for x := range r.rcMessages {
			prefix := ""
			color := colornames.White

			switch r.rcMessages[x].Flag {
			case Messages.ChequeredFlag:
				prefix = "🏁 "
			case Messages.GreenFlag:
				color = colornames.Green
				if strings.HasPrefix(r.rcMessages[x].Msg, "GREEN LIGHT") {
					prefix = "● "
				} else {
					prefix = "⚑ "
				}
			case Messages.YellowFlag:
				color = colornames.Yellow
				prefix = "⚑ "
			case Messages.DoubleYellowFlag:
				color = colornames.Yellow
				prefix = "⚑⚑ "
			case Messages.BlueFlag:
				color = colornames.Blue
				prefix = "⚑ "
			case Messages.RedFlag:
				color = colornames.Red
				if strings.HasPrefix(r.rcMessages[x].Msg, "RED LIGHT") {
					prefix = "● "
				} else {
					prefix = "⚑ "
				}
			case Messages.BlackAndWhite:
				color = colornames.White
				prefix = "⚑⚑"
			}

			if len(prefix) != 0 {
				msgs = append(msgs,
					giu.Style().SetStyleFloat(giu.StyleVarItemSpacing, 0).To(
						giu.Row(
							giu.Label(fmt.Sprintf("%s - ", r.rcMessages[x].Timestamp.In(r.dataSrc.CircuitTimezone()).
								Format("15:04:05"))),
							giu.Style().SetColor(giu.StyleColorText, color).To(giu.Label(prefix)),
							giu.Label(r.rcMessages[x].Msg))),
				)
			} else {
				msgs = append(msgs,
					giu.Label(
						fmt.Sprintf("%s - %s",
							r.rcMessages[x].Timestamp.In(r.dataSrc.CircuitTimezone()).
								Format("15:04:05"), r.rcMessages[x].Msg)))
			}
		}
	}
	r.rcMessagesLock.Unlock()

	return msgs
}
