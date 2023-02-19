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
	"github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/ungerik/go-cairo"
	"image/color"
	"math"
	"sort"
)

type gapperPlotInfo struct {
	color    color.RGBA
	name     string
	lapTimes []float64
	average  float64
	total    float64
	fastest  float64
	visible  bool
}

type gapperPlot struct {
	driverData           map[int]*gapperPlotInfo
	totalLaps            int
	driverNames          []string
	selectedDriver       int32
	selectedDriverNumber int
	yMin                 float64
	yMax                 float64

	visibleDriversSelect *gapperDriverDisplaySelectWidget

	plot            *plot
	yAxisPos        float64
	firstDriverY    float64
	xGap            float64
	yGap            float64
	endXPos         float64
	yAxisPosForZero float64
}

const NothingSelected = -1

func CreateGapperPlot() Panel {
	panel := &gapperPlot{
		driverData: map[int]*gapperPlotInfo{},
		totalLaps:  0,
	}
	panel.plot = createPlot(panel.drawBackground, panel.drawForeground)
	panel.visibleDriversSelect = &gapperDriverDisplaySelectWidget{
		plot:    panel.plot,
		drivers: []*gapperPlotInfo{},
	}
	return panel
}

func (g *gapperPlot) ProcessEventTime(data Messages.EventTime)                    {}
func (g *gapperPlot) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (g *gapperPlot) ProcessWeather(data Messages.Weather)                        {}
func (g *gapperPlot) ProcessRadio(data Messages.Radio)                            {}
func (g *gapperPlot) ProcessLocation(data Messages.Location)                      {}
func (g *gapperPlot) ProcessTelemetry(data Messages.Telemetry)                    {}
func (g *gapperPlot) Close()                                                      {}

func (g *gapperPlot) Type() Type { return GapperPlot }

func (g *gapperPlot) Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig) {
	g.driverData = map[int]*gapperPlotInfo{}
	g.totalLaps = 0
	g.driverNames = []string{}
	g.selectedDriver = NothingSelected
	g.selectedDriverNumber = NothingSelected
	g.yMin = math.MaxFloat64
	g.yMax = -math.MaxFloat64
	g.visibleDriversSelect.drivers = []*gapperPlotInfo{}
	g.visibleDriversSelect.visibleCount = 0
	g.plot.reset()
}

func (g *gapperPlot) ProcessDrivers(data Messages.Drivers) {
	for x := range data.Drivers {
		driver := &gapperPlotInfo{
			color:    data.Drivers[x].Color,
			name:     data.Drivers[x].ShortName,
			lapTimes: []float64{},
			fastest:  math.MaxFloat64,
			visible:  true,
		}
		g.driverData[data.Drivers[x].Number] = driver
		g.visibleDriversSelect.drivers = append(g.visibleDriversSelect.drivers, driver)

		g.driverNames = append(g.driverNames, data.Drivers[x].ShortName)
	}

	sort.Strings(g.driverNames)

	// After loading the drivers auto select the first driver in the list as a default
	g.selectedDriver = 0
	for num, driver := range g.driverData {
		if driver.name == g.driverNames[g.selectedDriver] {
			g.selectedDriverNumber = num
			break
		}
	}

	sort.Slice(g.visibleDriversSelect.drivers, func(i, j int) bool {
		return g.visibleDriversSelect.drivers[i].name < g.visibleDriversSelect.drivers[j].name
	})
	g.visibleDriversSelect.visibleCount = len(g.visibleDriversSelect.drivers)
}

func (g *gapperPlot) ProcessEvent(data Messages.Event) {
	if g.totalLaps == 0 {
		g.totalLaps = data.TotalLaps
		g.plot.refreshBackground()
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

		refreshBackground := false
		if g.selectedDriverNumber != NothingSelected && driverInfo.visible {
			// Update the yMin and yMax values for the whole chart
			baseline := g.driverData[g.selectedDriverNumber].fastest

			// If we don't have a fastest time yet then skip
			if baseline < math.MaxFloat64 {
				value := lapTimeSeconds - baseline

				if value < g.yMin {
					g.yMin = value
					refreshBackground = true
				}

				if value > g.yMax {
					g.yMax = value
					refreshBackground = true
				}
			}

			// If this is the selected driver and the first lap time for that driver then update
			// the Y min and max values for all other drivers who have data already
			if g.selectedDriverNumber == data.Number && len(driverInfo.lapTimes) == 1 {
				refreshBackground = g.refreshYMinMax() || refreshBackground
			}
		}

		// Background refresh will cause a foreground refresh so don't need both
		if refreshBackground {
			g.plot.refreshBackground()
		} else if driverInfo.visible {
			g.plot.refreshForeground()
		}
	}
}

func (g *gapperPlot) refreshYMinMax() bool {
	refreshBackground := false

	baseline := g.driverData[g.selectedDriverNumber].fastest

	// If we don't have a fastest time yet then skip
	if baseline < math.MaxFloat64 {
		return refreshBackground
	}

	g.yMin = math.MaxFloat64
	g.yMax = -math.MaxFloat64

	for key := range g.driverData {
		if !g.driverData[key].visible {
			continue
		}

		for _, time := range g.driverData[key].lapTimes {
			value := time - baseline
			if value < g.yMin {
				// Pad the value by 1
				g.yMin = value - 1
				refreshBackground = true
			}

			if value > g.yMax {
				// Pad the value by 1
				g.yMax = value + 1
				refreshBackground = true
			}
		}
	}
	return refreshBackground
}

func (g *gapperPlot) Draw(width int, height int) []giu.Widget {
	driverName := "<none>"
	if g.selectedDriver != NothingSelected {
		driverName = g.driverNames[g.selectedDriver]
	}

	return []giu.Widget{
		giu.Row(
			giu.Combo("Driver", driverName, g.driverNames, &g.selectedDriver).OnChange(func() {
				for num, driver := range g.driverData {
					if driver.name == g.driverNames[g.selectedDriver] {
						g.selectedDriverNumber = num
						if g.refreshYMinMax() {
							g.plot.refreshBackground()
						}
						break
					}
				}

				g.plot.refreshForeground()
			}).Size(100),
			g.visibleDriversSelect,
		),
		g.plot.draw(width-16, height-38),
	}
}

func (g *gapperPlot) drawBackground(dc *cairo.Surface) {
	width := float64(dc.GetWidth())
	height := float64(dc.GetHeight())

	// If no driver selected then draw nothing
	if g.selectedDriver == NothingSelected || math.Abs(g.driverData[g.selectedDriverNumber].fastest-math.MaxFloat64) < math.SmallestNonzeroFloat64 {
		// Black background
		dc.SetSourceRGB(0.0, 0.0, 0.0)
		dc.Rectangle(0, 0, width, height)
		dc.Fill()
		dc.Stroke()

		dc.SetSourceRGB(1.0, 1.0, 1.0)
		dc.MoveTo((width/2)-50, height/2)
		dc.ShowText("Waiting for data...")
		dc.Stroke()
		return
	}

	// Leave border all around the chart
	margin := 10.0
	// X location for Y axis
	g.yAxisPos = margin + 30
	// X pos end of X axis location
	g.endXPos = width - margin
	g.xGap = (g.endXPos - g.yAxisPos) / float64(g.totalLaps+1)
	yAxisLength := height - margin - margin
	// Gap per 1.0 increment on the y axis
	g.yGap = yAxisLength / (g.yMax + math.Abs(g.yMin))

	// Black background
	dc.SetSourceRGB(0.0, 0.0, 0.0)
	dc.Rectangle(0, 0, width, height)
	dc.Fill()
	dc.Stroke()

	// X Axis line - at 0 for the Y value
	dc.SetSourceRGB(0.0, 0.0, 1.0)
	g.yAxisPosForZero = margin + (g.yMax * g.yGap)
	dc.MoveTo(g.yAxisPos, g.yAxisPosForZero)
	dc.LineTo(width-margin, g.yAxisPosForZero)
	dc.Stroke()

	drawYAxis(
		dc,
		g.yAxisPos,
		margin,
		height-margin,
		g.yAxisPosForZero,
		0.0,
		g.yGap,
		1.0)
}

func (g *gapperPlot) drawForeground(dc *cairo.Surface) {
	if g.selectedDriverNumber == NothingSelected {
		return
	}

	baseline := g.driverData[g.selectedDriverNumber].fastest

	for key := range g.driverData {
		if !g.driverData[key].visible {
			continue
		}

		// If not enough lap times then draw nothing
		if len(g.driverData[key].lapTimes) == 0 {
			continue
		}

		// Draw line from start position to current position
		dc.SetSourceRGBA(floatColor(g.driverData[key].color))

		currentXPos := g.yAxisPos

		for x, lapTime := range g.driverData[key].lapTimes {
			value := lapTime - baseline
			yPos := g.yAxisPosForZero - (value * g.yGap)

			if x == 0 {
				dc.MoveTo(currentXPos, yPos)
			}

			dc.LineTo(currentXPos, yPos)

			currentXPos += g.xGap
		}
		dc.Stroke()
	}
}

type gapperDriverDisplaySelectWidget struct {
	id           string
	drivers      []*gapperPlotInfo
	plot         *plot
	visibleCount int
}

func (c *gapperDriverDisplaySelectWidget) Build() {
	redraw := false
	imgui.PushItemWidth(100)
	if imgui.BeginCombo("Display Data For", fmt.Sprintf("%d drivers", c.visibleCount)) {

		for x := range c.drivers {
			if imgui.Checkbox(c.drivers[x].name, &c.drivers[x].visible) {
				redraw = true
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	if redraw {
		// Background refresh will refresh the foreground too
		c.plot.refreshBackground()

		c.visibleCount = 0
		for x := range c.drivers {
			if c.drivers[x].visible {
				c.visibleCount++
			}
		}
	}
}
