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
	"image/color"
	"sort"
	"sync"
)

type info struct {
	color     color.RGBA
	number    int
	name      string
	positions []float64
}

type racePosition struct {
	driverData  map[int]*info
	orderedData []*info
	totalLaps   int

	linesLock sync.Mutex
	lines     []giu.PlotWidget
}

func CreateRacePosition() Panel {
	return &racePosition{
		lines:       []giu.PlotWidget{},
		driverData:  map[int]*info{},
		orderedData: []*info{},
		totalLaps:   0,
	}
}

func (r *racePosition) ProcessEventTime(data Messages.EventTime)                    {}
func (r *racePosition) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (r *racePosition) ProcessWeather(data Messages.Weather)                        {}
func (r *racePosition) ProcessRadio(data Messages.Radio)                            {}
func (r *racePosition) ProcessLocation(data Messages.Location)                      {}
func (r *racePosition) ProcessTelemetry(data Messages.Telemetry)                    {}
func (r *racePosition) Close()                                                      {}

func (r *racePosition) Type() Type { return RacePosition }

func (r *racePosition) Init(dataSrc f1gopherlib.F1GopherLib) {
	// Clear previous session data
	r.lines = []giu.PlotWidget{}
	r.driverData = map[int]*info{}
	r.orderedData = []*info{}
	r.totalLaps = 0
}

func (r *racePosition) ProcessDrivers(data Messages.Drivers) {
	for x := range data.Drivers {
		driverInfo := &info{
			color:     data.Drivers[x].Color,
			number:    data.Drivers[x].Number,
			name:      data.Drivers[x].ShortName,
			positions: []float64{float64(data.Drivers[x].StartPosition)},
		}

		r.driverData[data.Drivers[x].Number] = driverInfo
		r.orderedData = append(r.orderedData, driverInfo)
	}

	sort.Slice(r.orderedData, func(i, j int) bool {
		return r.orderedData[i].positions[0] < r.orderedData[j].positions[0]
	})

	tmpLines := []giu.PlotWidget{}
	for x := range r.orderedData {
		tmpLines = append(tmpLines, giu.PlotLine(r.orderedData[x].name, r.orderedData[x].positions))
	}

	r.linesLock.Lock()
	r.lines = tmpLines
	r.linesLock.Unlock()
}

func (r *racePosition) ProcessEvent(data Messages.Event) {
	if r.totalLaps == 0 {
		r.totalLaps = data.TotalLaps
	}
}

func (r *racePosition) ProcessTiming(data Messages.Timing) {
	driverInfo := r.driverData[data.Number]

	count := len(driverInfo.positions)
	if count == data.Lap {
		driverInfo.positions = append(driverInfo.positions, float64(data.Position))

		tmpLines := []giu.PlotWidget{}
		for x := range r.orderedData {
			tmpLines = append(tmpLines, giu.PlotLine(r.orderedData[x].name, r.orderedData[x].positions))
		}

		r.linesLock.Lock()
		r.lines = tmpLines
		r.linesLock.Unlock()
	}
}

func (r *racePosition) Draw(width int, height int) []giu.Widget {
	r.linesLock.Lock()
	defer r.linesLock.Unlock()

	return []giu.Widget{
		giu.Plot("Race Position").Plots(r.lines...).
			Size(width-16, height-16).
			AxisLimits(0, float64(r.totalLaps), 0, float64(len(r.lines)), giu.ConditionAppearing),
	}
}
