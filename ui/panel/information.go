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
	"sync"
	"time"

	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
)

type information struct {
	exit          func()
	dataSrc       f1gopherlib.F1GopherLib
	isLiveSession bool

	event         Messages.Event
	eventLock     sync.Mutex
	eventTime     time.Time
	remainingTime time.Duration
}

func CreateInformation(exit func(), isLiveSession bool) Panel {
	return &information{
		exit:          exit,
		isLiveSession: isLiveSession,
	}
}

func (i *information) ProcessDrivers(data Messages.Drivers)                        {}
func (i *information) ProcessTiming(data Messages.Timing)                          {}
func (i *information) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (i *information) ProcessWeather(data Messages.Weather)                        {}
func (i *information) ProcessRadio(data Messages.Radio)                            {}
func (i *information) ProcessLocation(data Messages.Location)                      {}
func (i *information) ProcessTelemetry(data Messages.Telemetry)                    {}
func (i *information) Close()                                                      {}

func (i *information) Type() Type { return Info }

func (i *information) Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig) {
	i.dataSrc = dataSrc

	// Clear previous session data
	i.event = Messages.Event{}
	i.remainingTime = 0
}

func (i *information) ProcessEventTime(data Messages.EventTime) {
	i.eventTime = data.Timestamp
	i.remainingTime = data.Remaining
}

func (i *information) ProcessEvent(data Messages.Event) {
	i.eventLock.Lock()
	i.event = data
	i.eventLock.Unlock()
}

func (i *information) Draw(width int, height int) []giu.Widget {

	pauseTxt := "Pause"
	if i.dataSrc.IsPaused() {
		pauseTxt = "Resume"
	}

	// Once the remaining time counter is not zero it has started counting down because the session has started
	hasStarted := i.remainingTime != 0 && !i.isLiveSession

	panelWidgets := []giu.Widget{
		giu.Row(
			i.infoWidgets(),
			giu.Button("Skip 5 Seconds").OnClick(func() {
				i.dataSrc.IncrementTime(time.Second * 5)
			}),
			giu.Button("Skip Minute").OnClick(func() {
				i.dataSrc.IncrementTime(time.Minute * 1)
			}),
			giu.Button("Skip To Start").OnClick(func() {
				i.dataSrc.SkipToSessionStart()
			}).Disabled(hasStarted),
			giu.Button(pauseTxt).OnClick(func() {
				i.dataSrc.TogglePause()
			}),
			giu.Button("Back").OnClick(func() {
				// Do this on another routine so this one can exit and stop drawing releasing the waitgroup that
				// exit will wait for
				go func() { i.exit() }()
			})),
	}

	return panelWidgets
}

func (i *information) infoWidgets() *giu.RowWidget {
	hour := int(i.remainingTime.Seconds() / 3600)
	minute := int(i.remainingTime.Seconds()/60) % 60
	second := int(i.remainingTime.Seconds()) % 60
	remaining := fmt.Sprintf("%d:%02d:%02d", hour, minute, second)

	i.eventLock.Lock()
	defer i.eventLock.Unlock()

	widgets := []giu.Widget{
		giu.Label(fmt.Sprintf(
			"%s: %v, Track Time: %v, Status:",
			i.dataSrc.Name(),
			i.event.Type.String(),
			i.eventTime.In(i.dataSrc.CircuitTimezone()).Format("2006-01-02 15:04:05"))),
		giu.Style().SetColor(giu.StyleColorText, sessionStatusColor(i.event.Status)).To(
			giu.Label(i.event.TrackStatus.String())),
	}

	// These are only relevant for a race session
	if i.event.Type == Messages.Race || i.event.Type == Messages.Sprint {
		widgets = append(widgets,
			giu.Label(fmt.Sprintf(", DRS: %v, Safety Car:",
				i.event.DRSEnabled.String())))

		widgets = append(widgets,
			giu.Style().SetColor(giu.StyleColorText, safetyCarFormat(i.event.SafetyCar)).To(
				giu.Label(i.event.SafetyCar.String())))

		widgets = append(widgets,
			giu.Label(fmt.Sprintf(", Lap: %d/%d", i.event.CurrentLap, i.event.TotalLaps)))
	}

	// If the session hasn't started then display a count down instead of the remaining time
	if i.eventTime.Before(i.dataSrc.SessionStart()) {
		widgets = append(widgets,
			giu.Label(fmt.Sprintf(", Session Starts in: %s", fmtCountdown(i.dataSrc.SessionStart().Sub(i.eventTime)))))
	} else {
		widgets = append(widgets,
			giu.Label(fmt.Sprintf(", Remaining: %s", remaining)))
	}

	widgets = append(widgets, giu.Style().SetColor(giu.StyleColorText, trackStatusColor(i.event.TrackStatus)).To(
		giu.Label(flagCharacter())))

	return giu.Row(widgets...)
}

func fmtCountdown(d time.Duration) string {
	milliseconds := d.Milliseconds()

	if milliseconds == 0 {
		return ""
	}

	minutes := milliseconds / (1000 * 60)
	milliseconds -= minutes * 60 * 1000
	seconds := milliseconds / 1000
	milliseconds -= seconds * 1000

	if minutes < 0 {
		minutes = -minutes
	}
	if seconds < 0 {
		seconds = -seconds
	}

	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
