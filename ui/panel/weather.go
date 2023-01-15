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
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"golang.org/x/image/colornames"
	"sync"
	"sync/atomic"
)

type weather struct {
	data        Messages.Weather
	dataLock    sync.Mutex
	dataChanged atomic.Bool

	cachedUI []giu.Widget
}

func CreateWeather() Panel {
	return &weather{
		cachedUI: make([]giu.Widget, 0),
	}
}

func (w *weather) Init(dataSrc f1gopherlib.F1GopherLib) {
	// Clear previous data
	w.cachedUI = make([]giu.Widget, 0)
	w.data = Messages.Weather{}
}

func (w *weather) ProcessDrivers(data Messages.Drivers)                        {}
func (w *weather) ProcessTiming(data Messages.Timing)                          {}
func (w *weather) ProcessEventTime(data Messages.EventTime)                    {}
func (w *weather) ProcessEvent(data Messages.Event)                            {}
func (w *weather) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (w *weather) ProcessRadio(data Messages.Radio)                            {}
func (w *weather) ProcessLocation(data Messages.Location)                      {}
func (w *weather) ProcessTelemetry(data Messages.Telemetry)                    {}
func (w *weather) Close()                                                      {}

func (w *weather) Type() Type { return Weather }

func (w *weather) ProcessWeather(data Messages.Weather) {
	w.dataLock.Lock()
	w.data = data
	w.dataLock.Unlock()
	w.dataChanged.Store(true)
}

func (w *weather) Draw(width int, height int) []giu.Widget {
	if w.dataChanged.CompareAndSwap(true, false) {
		w.dataChanged.Store(false)
		w.cachedUI = w.widgets()
	}

	return w.cachedUI
}

func (w *weather) widgets() []giu.Widget {
	widgets := make([]giu.Widget, 0)

	w.dataLock.Lock()

	if w.data.Rainfall {
		widgets = append(widgets, giu.Style().SetColor(giu.StyleColorText, colornames.Cornflowerblue).To(giu.Label("Rain")))
	} else {
		widgets = append(widgets, giu.Label("No rain"))
	}
	widgets = append(widgets, giu.Labelf("Air Temp: %.1f°C", w.data.AirTemp))
	widgets = append(widgets, giu.Labelf("Track Temp: %.1f°C", w.data.TrackTemp))
	widgets = append(widgets, giu.Labelf("Wind Speed: %.0f", w.data.WindSpeed))
	widgets = append(widgets, giu.Labelf("Wind Direction: %.0f", w.data.WindDirection))
	widgets = append(widgets, giu.Labelf("Air Pressure: %.1f", w.data.AirPressure))
	widgets = append(widgets, giu.Labelf("Humidity: %.1f%%", w.data.Humidity))

	w.dataLock.Unlock()

	return widgets
}
