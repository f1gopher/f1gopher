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
	"github.com/ungerik/go-cairo"
	"image/color"
	"sort"
)

type info struct {
	color     color.RGBA
	number    int
	name      string
	positions []int
}

type racePosition struct {
	driverData  map[int]*info
	orderedData []*info
	totalLaps   int

	plot         *plot
	yAxisPos     float64
	firstDriverY float64
	xGap         float64
	yGap         float64
	endXPos      float64
}

func CreateRacePosition() Panel {
	panel := &racePosition{
		driverData:  map[int]*info{},
		orderedData: []*info{},
		totalLaps:   0,
	}
	panel.plot = createPlot(panel.drawBackground, panel.drawForeground)

	return panel
}

func (r *racePosition) ProcessEventTime(data Messages.EventTime)                    {}
func (r *racePosition) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (r *racePosition) ProcessWeather(data Messages.Weather)                        {}
func (r *racePosition) ProcessRadio(data Messages.Radio)                            {}
func (r *racePosition) ProcessLocation(data Messages.Location)                      {}
func (r *racePosition) ProcessTelemetry(data Messages.Telemetry)                    {}
func (r *racePosition) Close()                                                      {}

func (r *racePosition) Type() Type { return RacePosition }

func (r *racePosition) Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig) {
	// Clear previous session data
	r.driverData = map[int]*info{}
	r.orderedData = []*info{}
	r.totalLaps = 0
	r.plot.reset()
}

func (r *racePosition) ProcessDrivers(data Messages.Drivers) {
	for x := range data.Drivers {
		driverInfo := &info{
			color:     data.Drivers[x].Color,
			number:    data.Drivers[x].Number,
			name:      data.Drivers[x].ShortName,
			positions: []int{data.Drivers[x].StartPosition},
		}

		r.driverData[data.Drivers[x].Number] = driverInfo
		r.orderedData = append(r.orderedData, driverInfo)
	}

	sort.Slice(r.orderedData, func(i, j int) bool {
		return r.orderedData[i].positions[0] < r.orderedData[j].positions[0]
	})

	r.plot.refreshBackground()
}

func (r *racePosition) ProcessEvent(data Messages.Event) {
	if r.totalLaps == 0 {
		r.totalLaps = data.TotalLaps
		r.plot.refreshBackground()
	}
}

func (r *racePosition) ProcessTiming(data Messages.Timing) {
	driverInfo, exists := r.driverData[data.Number]

	if !exists {
		return
	}

	count := len(driverInfo.positions)
	if count == data.Lap {
		driverInfo.positions = append(driverInfo.positions, data.Position)
		r.plot.refreshForeground()
	}
}

func (r *racePosition) Draw(width int, height int) []giu.Widget {
	return []giu.Widget{
		r.plot.draw(width-16, height-16),
	}
}

func (r *racePosition) drawBackground(dc *cairo.Surface) {
	width := float64(dc.GetWidth())
	height := float64(dc.GetHeight())

	// Leave border of 20px all around the chart
	margin := 10.0
	// X location for Y axis
	r.yAxisPos = margin + 30
	// X pos end of X axis location
	r.endXPos = width - margin
	r.xGap = (r.endXPos - r.yAxisPos) / float64(r.totalLaps+1)

	// Black background
	dc.SetSourceRGB(0.0, 0.0, 0.0)
	dc.Rectangle(0, 0, width, height)
	dc.Fill()
	dc.Stroke()

	// X & Y Axis lines
	dc.SetSourceRGB(0.0, 0.0, 1.0)
	dc.MoveTo(r.yAxisPos, margin)
	dc.LineTo(r.yAxisPos, height-margin)
	dc.LineTo(width-margin, height-margin)
	dc.Stroke()

	// Driver names for Y axis
	numDrivers := len(r.orderedData)
	yAxisLength := height - margin - margin
	r.yGap = yAxisLength / float64(numDrivers)
	r.firstDriverY = margin + r.yGap/2
	currentYPos := r.firstDriverY + 2 // add half of text height to centre label

	for x := range r.orderedData {
		dc.SetSourceRGBA(floatColor(r.orderedData[x].color))
		dc.MoveTo(margin, currentYPos)
		dc.ShowText(r.orderedData[x].name)
		currentYPos += r.yGap
	}
}

func (r *racePosition) drawForeground(dc *cairo.Surface) {
	currentStartYPos := r.firstDriverY

	// Draw a position line for each driver
	for x := range r.orderedData {
		currentXPos := r.yAxisPos

		// If not enough positions then draw nothing
		if len(r.orderedData[x].positions) <= 1 {
			continue
		}

		// Draw line from start position to current position
		dc.SetSourceRGBA(floatColor(r.orderedData[x].color))
		dc.MoveTo(currentXPos, currentStartYPos)
		currentStartYPos += r.yGap

		for _, pos := range r.orderedData[x].positions {
			currentXPos += r.xGap
			dc.LineTo(currentXPos, r.firstDriverY+(float64(pos-1)*r.yGap))
		}
		dc.Stroke()
	}
}
