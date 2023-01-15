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
	"math"
	"sort"
	"sync"
)

type gapperPlotInfo struct {
	color    color.RGBA
	name     string
	lapTimes []float64
	average  float64
	total    float64
	fastest  float64
}

type gapperPlot struct {
	driverData           map[int]*gapperPlotInfo
	totalLaps            int
	driverNames          []string
	selectedDriver       int32
	selectedDriverNumber int
	yMin                 float64
	yMax                 float64

	linesLock sync.Mutex
	lines     []giu.PlotWidget
}

func CreateGapperPlot() Panel {
	return &gapperPlot{
		driverData: map[int]*gapperPlotInfo{},

		totalLaps: 0,
	}
}

func (g *gapperPlot) ProcessEventTime(data Messages.EventTime)                    {}
func (g *gapperPlot) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (g *gapperPlot) ProcessWeather(data Messages.Weather)                        {}
func (g *gapperPlot) ProcessRadio(data Messages.Radio)                            {}
func (g *gapperPlot) ProcessLocation(data Messages.Location)                      {}
func (g *gapperPlot) ProcessTelemetry(data Messages.Telemetry)                    {}
func (g *gapperPlot) Close()                                                      {}

func (g *gapperPlot) Type() Type { return GapperPlot }

func (g *gapperPlot) Init(dataSrc f1gopherlib.F1GopherLib) {
	g.driverData = map[int]*gapperPlotInfo{}
	g.totalLaps = 0
	g.driverNames = []string{}
	g.selectedDriver = -1
	g.selectedDriverNumber = -1
	g.yMin = 0.0
	g.yMax = 0.0
}

func (g *gapperPlot) ProcessDrivers(data Messages.Drivers) {
	for x := range data.Drivers {
		g.driverData[data.Drivers[x].Number] = &gapperPlotInfo{
			color:    data.Drivers[x].Color,
			name:     data.Drivers[x].ShortName,
			lapTimes: []float64{},
			fastest:  math.MaxFloat64,
		}

		g.driverNames = append(g.driverNames, data.Drivers[x].ShortName)
		sort.Strings(g.driverNames)
	}
}

func (g *gapperPlot) ProcessEvent(data Messages.Event) {
	if g.totalLaps == 0 {
		g.totalLaps = data.TotalLaps
	}
}

func (g *gapperPlot) ProcessTiming(data Messages.Timing) {
	// TODO - when the safety car comes out we don't get a lap time - brazil 2022
	// TODO - we don't get a lap time for the first lap - try calculate one in the lib?
	if data.LastLap == 0 {
		return
	}

	driverInfo := g.driverData[data.Number]

	// We don't get a lap time for the first lap
	if data.Lap == len(driverInfo.lapTimes)+2 {
		lapTimeSeconds := data.LastLap.Seconds()

		driverInfo.lapTimes = append(driverInfo.lapTimes, lapTimeSeconds)
		driverInfo.total += lapTimeSeconds
		driverInfo.average = driverInfo.total / float64(len(driverInfo.lapTimes))
		driverInfo.fastest = math.Min(driverInfo.fastest, lapTimeSeconds)

		g.redraw()
	}
}

func (g *gapperPlot) Draw(width int, height int) []giu.Widget {
	g.linesLock.Lock()
	defer g.linesLock.Unlock()

	driverName := "<none>"
	if g.selectedDriver != -1 {
		driverName = g.driverNames[g.selectedDriver]
	}

	return []giu.Widget{
		giu.Combo("Driver", driverName, g.driverNames, &g.selectedDriver).OnChange(func() {
			for num, driver := range g.driverData {
				if driver.name == g.driverNames[g.selectedDriver] {
					g.selectedDriverNumber = num
					break
				}
			}

			// TODO - set flag to be more efficient/responsive
			g.redraw()
		}),

		giu.Plot("Gapper Plot").Plots(g.lines...).
			Size(width-16, height-36).
			AxisLimits(
				0,
				float64(g.totalLaps),
				g.yMin-1,
				g.yMax+1,
				giu.ConditionAppearing),
	}
}

func (g *gapperPlot) redraw() {
	if g.selectedDriverNumber != -1 {
		baseline := g.driverData[g.selectedDriverNumber].fastest

		yMin := math.MaxFloat64
		yMax := math.SmallestNonzeroFloat64
		tmpLines := []giu.PlotWidget{}
		for x := range g.driverData {
			values := []float64{}

			for _, t := range g.driverData[x].lapTimes {
				val := t - baseline
				values = append(values, val)

				yMin = math.Min(yMin, val)
				yMax = math.Max(yMax, val)
			}

			tmpLines = append(tmpLines, giu.PlotLine(g.driverData[x].name, values))
		}

		g.yMin = yMin
		g.yMax = yMax

		g.linesLock.Lock()
		g.lines = tmpLines
		g.linesLock.Unlock()
	}
}
