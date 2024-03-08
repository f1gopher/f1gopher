// F1Gopher - Copyright (C) 2023 f1gopher
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package panel

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"golang.org/x/image/colornames"
)

func flagCharacter() string {
	// Not been able to get the unicode flag character to work on Linux
	switch runtime.GOOS {
	case "linux":
		return "▛"
	default:
		return "⚑"
	}
}

type raceControlMessages struct {
	dataSrc        f1gopherlib.F1GopherLib
	rcMessages     []Messages.RaceControlMessage
	rcMessagesLock sync.Mutex
	dataChanged    atomic.Bool
	scrollToBottom *ScrollToBottomWidget

	cachedUI []giu.Widget
}

func CreateRaceControlMessages() Panel {
	return &raceControlMessages{
		rcMessages:     make([]Messages.RaceControlMessage, 0),
		scrollToBottom: &ScrollToBottomWidget{id: "scroll to bottom"},
	}
}

func (r *raceControlMessages) ProcessDrivers(data Messages.Drivers)     {}
func (r *raceControlMessages) ProcessTiming(data Messages.Timing)       {}
func (r *raceControlMessages) ProcessEventTime(data Messages.EventTime) {}
func (r *raceControlMessages) ProcessEvent(data Messages.Event)         {}
func (r *raceControlMessages) ProcessWeather(data Messages.Weather)     {}
func (r *raceControlMessages) ProcessRadio(data Messages.Radio)         {}
func (r *raceControlMessages) ProcessLocation(data Messages.Location)   {}
func (r *raceControlMessages) ProcessTelemetry(data Messages.Telemetry) {}
func (r *raceControlMessages) Close()                                   {}

func (r *raceControlMessages) Type() Type { return RaceControlMessages }

func (r *raceControlMessages) Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig) {
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

		// The first time we redraw after a new message scroll to the bottom
		return append(r.cachedUI, r.scrollToBottom)
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
			case Messages.GreenFlag:
				color = colornames.Green
				if strings.HasPrefix(r.rcMessages[x].Msg, "GREEN LIGHT") {
					prefix = "● "
				} else {
					prefix = flagCharacter() + " "
				}
			case Messages.YellowFlag:
				color = colornames.Yellow
				prefix = flagCharacter() + " "
			case Messages.DoubleYellowFlag:
				color = colornames.Yellow
				prefix = flagCharacter() + flagCharacter() + " "
			case Messages.BlueFlag:
				color = colornames.Blue
				prefix = flagCharacter() + " "
			case Messages.RedFlag:
				color = colornames.Red
				if strings.HasPrefix(r.rcMessages[x].Msg, "RED LIGHT") {
					prefix = "● "
				} else {
					prefix = flagCharacter() + " "
				}
			case Messages.BlackAndWhite:
				color = colornames.White
				prefix = flagCharacter() + flagCharacter()
			}

			if len(prefix) != 0 {
				msgs = append(msgs,
					giu.Style().SetColor(giu.StyleColorText, color).To(giu.Label(fmt.Sprintf("%s %s - %s", r.rcMessages[x].Timestamp.In(r.dataSrc.CircuitTimezone()).
						Format("15:04:05"), prefix, r.rcMessages[x].Msg)).Wrapped(true)))
			} else {
				msgs = append(msgs,
					giu.Label(
						fmt.Sprintf("%s - %s",
							r.rcMessages[x].Timestamp.In(r.dataSrc.CircuitTimezone()).
								Format("15:04:05"), r.rcMessages[x].Msg)).Wrapped(true))
			}
		}
	}
	r.rcMessagesLock.Unlock()

	return msgs
}

type ScrollToBottomWidget struct {
	id string
}

func (c *ScrollToBottomWidget) Build() {
	imgui.SetScrollHereY(1.0)
}
