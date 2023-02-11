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
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/ungerik/go-cairo"
	"image/color"
	"sort"
	"time"
)

type telemetryInfo struct {
	name   string
	color  color.RGBA
	number int
}

type channelConfig struct {
	name    string
	enabled bool

	top                float64
	bottom             float64
	gap                float64
	height             float64
	maxValue           float64
	majorTickIncrement float64

	colorR float64
	colorG float64
	colorB float64

	value        func(int) float64
	summaryValue func(int) string
}

type telemetry struct {
	dataSrc f1gopherlib.F1GopherLib

	data                 map[int]*telemetryInfo
	driverNames          []string
	selectedDriver       int32
	selectedDriverNumber int

	// TODO - use circular buffer of fixed size?
	rpm      *circularBuffer[int16]
	speed    *circularBuffer[float32]
	gear     *circularBuffer[byte]
	throttle *circularBuffer[float32]
	brake    *circularBuffer[float32]
	drs      *circularBuffer[bool]
	time     *circularBuffer[time.Time]

	currentTime     time.Time
	circuitTimezone *time.Location

	channelSelect *channelDisplaySelectWidget

	plot         *plot
	yAxisPos     float64
	xGap         float64
	endXPos      float64
	yBottom      float64
	xWidth       float64
	summaryWidth float64

	channels []channelConfig
}

func CreateTelemetry() Panel {
	const bufferSize = 150

	panel := &telemetry{
		data: map[int]*telemetryInfo{},
		channelSelect: &channelDisplaySelectWidget{
			id: "telemetryChannelSelect",
		},
		rpm:      createCircularBuffer[int16](bufferSize),
		speed:    createCircularBuffer[float32](bufferSize),
		gear:     createCircularBuffer[byte](bufferSize),
		throttle: createCircularBuffer[float32](bufferSize),
		brake:    createCircularBuffer[float32](bufferSize),
		drs:      createCircularBuffer[bool](bufferSize),
		time:     createCircularBuffer[time.Time](bufferSize),
	}
	panel.plot = createPlot(panel.drawBackground, panel.drawForeground)
	panel.channels = []channelConfig{
		{
			name:               "Brake",
			enabled:            true,
			maxValue:           100.0,
			majorTickIncrement: 20.0,
			colorR:             1.0,
			colorG:             0.0,
			colorB:             0.0,
			value:              func(x int) float64 { return float64(panel.brake.get(x)) },
			summaryValue:       func(x int) string { return fmt.Sprintf("%.0f%%", panel.brake.get(x)) },
		},
		{
			name:               "DRS",
			enabled:            true,
			maxValue:           1.0,
			majorTickIncrement: 10.0,
			colorR:             0.5,
			colorG:             0.5,
			colorB:             0.5,
			value: func(x int) float64 {
				if panel.drs.get(x) {
					return 1.0
				}
				return 0.0
			},
			summaryValue: func(x int) string {
				if panel.drs.get(x) {
					return "Open"
				}
				return "Closed"
			},
		},
		{
			name:               "Gear",
			enabled:            true,
			maxValue:           8.0,
			majorTickIncrement: 1.0,
			colorR:             0.0,
			colorG:             1.0,
			colorB:             1.0,
			value:              func(x int) float64 { return float64(panel.gear.get(x)) },
			summaryValue:       func(x int) string { return fmt.Sprintf("%d", panel.gear.get(x)) },
		},
		{
			name:               "RPM",
			enabled:            true,
			maxValue:           16000.0,
			majorTickIncrement: 3000.0,
			colorR:             1.0,
			colorG:             1.0,
			colorB:             0.0,
			value:              func(x int) float64 { return float64(panel.rpm.get(x)) },
			summaryValue:       func(x int) string { return fmt.Sprintf("%d", panel.rpm.get(x)) },
		},
		{
			name:               "Speed",
			enabled:            true,
			maxValue:           380.0,
			majorTickIncrement: 50.0,
			colorR:             0.0,
			colorG:             1.0,
			colorB:             0.0,
			value:              func(x int) float64 { return float64(panel.speed.get(x)) },
			summaryValue:       func(x int) string { return fmt.Sprintf("%.0f km/hr", panel.speed.get(x)) },
		},
		{
			name:               "Throttle",
			enabled:            true,
			maxValue:           100.0,
			majorTickIncrement: 20.0,
			colorR:             0.0,
			colorG:             0.0,
			colorB:             1.0,
			value:              func(x int) float64 { return float64(panel.throttle.get(x)) },
			summaryValue:       func(x int) string { return fmt.Sprintf("%.0f%%", panel.throttle.get(x)) },
		},
	}
	panel.channelSelect.plot = panel.plot
	panel.channelSelect.channels = &panel.channels

	return panel
}

func (t *telemetry) ProcessTiming(data Messages.Timing)                          {}
func (t *telemetry) ProcessEvent(data Messages.Event)                            {}
func (t *telemetry) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (t *telemetry) ProcessWeather(data Messages.Weather)                        {}
func (t *telemetry) ProcessRadio(data Messages.Radio)                            {}
func (t *telemetry) ProcessLocation(data Messages.Location)                      {}

func (t *telemetry) Type() Type { return Telemetry }

func (t *telemetry) Init(dataSrc f1gopherlib.F1GopherLib) {
	t.dataSrc = dataSrc
	t.circuitTimezone = dataSrc.CircuitTimezone()
	t.data = map[int]*telemetryInfo{}
	t.driverNames = nil
	t.selectedDriver = NothingSelected
	t.selectedDriverNumber = NothingSelected

	t.rpm.reset()
	t.speed.reset()
	t.gear.reset()
	t.throttle.reset()
	t.brake.reset()
	t.drs.reset()
	t.time.reset()

	t.plot.reset()
}

func (t *telemetry) Close() {
	t.rpm = nil
	t.speed = nil
	t.gear = nil
	t.throttle = nil
	t.brake = nil
	t.drs = nil
	t.time = nil
}

func (t *telemetry) ProcessDrivers(data Messages.Drivers) {
	for x := range data.Drivers {
		driver := &telemetryInfo{
			name:   data.Drivers[x].ShortName,
			number: data.Drivers[x].Number,
			color:  data.Drivers[x].Color,
		}
		t.data[data.Drivers[x].Number] = driver
		t.driverNames = append(t.driverNames, driver.name)
	}

	sort.Strings(t.driverNames)
}

func (t *telemetry) ProcessTelemetry(data Messages.Telemetry) {
	if t.selectedDriverNumber != data.DriverNumber {
		return
	}

	t.rpm.add(data.RPM)
	t.speed.add(data.Speed)
	t.gear.add(data.Gear)
	t.throttle.add(data.Throttle)
	t.brake.add(data.Brake)
	t.drs.add(data.DRS)
	t.time.add(data.Timestamp)

	if t.time.count() == 1 {
		t.plot.refreshBackground()
	} else {
		t.plot.refreshForeground()
	}
}

func (t *telemetry) ProcessEventTime(data Messages.EventTime) {
	t.currentTime = data.Timestamp
}

func (t *telemetry) Draw(width int, height int) []giu.Widget {
	driverName := "<none>"
	if t.selectedDriver != NothingSelected {
		driverName = t.driverNames[t.selectedDriver]
	}

	return []giu.Widget{
		giu.Row(
			giu.Combo("Driver", driverName, t.driverNames, &t.selectedDriver).OnChange(func() {
				for num, driver := range t.data {
					if driver.name == t.driverNames[t.selectedDriver] {
						t.selectedDriverNumber = num

						// Request data from newly selected driver only
						t.dataSrc.SelectTelemetrySources([]int{num})

						// Clea existing data
						t.rpm.reset()
						t.speed.reset()
						t.gear.reset()
						t.throttle.reset()
						t.brake.reset()
						t.drs.reset()
						t.time.reset()

						t.plot.refreshBackground()
						break
					}
				}
			}).Size(100),
			t.channelSelect,
		),
		t.plot.draw(width-16, height-38),
	}
}

func (t *telemetry) drawBackground(dc *cairo.Surface) {
	width := float64(dc.GetWidth())
	height := float64(dc.GetHeight())

	// If no driver selected then draw nothing
	if t.selectedDriver == NothingSelected {
		// Black background
		dc.SetSourceRGB(0.0, 0.0, 0.0)
		dc.Rectangle(0, 0, width, height)
		dc.Fill()
		dc.Stroke()

		dc.SetSourceRGB(1.0, 1.0, 1.0)
		dc.MoveTo((width/2)-50, height/2)
		dc.ShowText("Select Driver")
		return
	}

	// Leave border all around the chart
	margin := 10.0
	t.yBottom = height - margin - 20
	yAxisLength := t.yBottom - margin

	// Update top and bottom positions for each channel
	enabledChannels := 0.0
	for x := range t.channels {
		if !t.channels[x].enabled {
			continue
		}

		enabledChannels++
	}

	top := margin
	for x := range t.channels {
		if !t.channels[x].enabled {
			continue
		}

		t.channels[x].top = top

		// More accurate to just make the last one match the yBottom
		if x == len(t.channels)-1 {
			t.channels[x].bottom = t.yBottom
		} else {
			plotHeight := yAxisLength * (1.0 / enabledChannels)
			t.channels[x].bottom = (t.channels[x].top + plotHeight) - 1
		}

		top = t.channels[x].bottom + 1

		t.channels[x].gap = (t.channels[x].bottom - t.channels[x].top) / t.channels[x].maxValue
	}

	t.summaryWidth = 60
	// X location for Y axis
	t.yAxisPos = margin + 40
	// X pos end of X axis location
	t.endXPos = width - margin - t.summaryWidth
	t.xWidth = t.endXPos - t.yAxisPos

	// Black background
	dc.SetSourceRGB(0.0, 0.0, 0.0)
	dc.Rectangle(0, 0, width, height)
	dc.Fill()
	dc.Stroke()

	// X Axis line
	dc.SetSourceRGB(0.0, 0.0, 1.0)
	dc.MoveTo(t.yAxisPos, t.yBottom)
	dc.LineTo(t.endXPos, t.yBottom)
	dc.Stroke()

	for x := range t.channels {
		if !t.channels[x].enabled {
			continue
		}

		dc.SetSourceRGB(t.channels[x].colorR, t.channels[x].colorG, t.channels[x].colorB)

		drawYAxis(
			dc,
			t.yAxisPos,
			t.channels[x].top,
			t.channels[x].bottom,
			t.channels[x].bottom,
			0.0,
			t.channels[x].gap,
			t.channels[x].majorTickIncrement)
	}
}

func (t *telemetry) drawForeground(dc *cairo.Surface) {
	// If no driver selected do nothing
	if t.selectedDriver == NothingSelected || t.time.count() == 0 {
		return
	}

	xAxisSecondsLength := 120.0
	xPixelsPerSecond := t.xWidth / xAxisSecondsLength

	// End is a whole seconds
	endTime := t.currentTime.Round(time.Second)
	startTime := endTime.Add(-(time.Second * time.Duration(xAxisSecondsLength)))
	startOffset := xPixelsPerSecond * endTime.Sub(t.currentTime).Seconds()

	// Draw X axis marks and values
	dc.SetSourceRGB(0.0, 0.0, 1.0)
	currentTime := startTime
	xPos := t.yAxisPos + startOffset
	for xPos < t.endXPos {
		if currentTime.Second()%10 != 0 {
			xPos += xPixelsPerSecond
			currentTime = currentTime.Add(time.Second)
			continue
		}

		dc.MoveTo(xPos, t.yBottom)
		dc.LineTo(xPos, t.yBottom+15.0)

		dc.MoveTo(xPos-20, t.yBottom+25)
		dc.ShowText(currentTime.In(t.circuitTimezone).Format("15:04:05"))

		xPos += xPixelsPerSecond
		currentTime = currentTime.Add(time.Second)
	}
	dc.Stroke()

	// Draw data channel
	last := t.time.count() - 1
	for _, channel := range t.channels {
		if !channel.enabled {
			continue
		}

		dc.SetSourceRGB(channel.colorR, channel.colorG, channel.colorB)

		currentTime = startTime
		xPos = t.yAxisPos + startOffset
		first := true

		// Data line
		for x := 0; x < t.time.count(); x++ {
			// Don't draw every point. At most one per pixel
			if t.time.get(x).Before(currentTime) {
				continue
			}

			if first {
				dc.MoveTo(xPos, channel.bottom-channel.gap*channel.value(x))
				first = false
			}

			dc.LineTo(xPos, channel.bottom-channel.gap*channel.value(x))
			currentTime = currentTime.Add(time.Second * 1.0)
			xPos += xPixelsPerSecond
		}

		// Summary
		center := channel.top + ((channel.bottom - channel.top) / 2)
		dc.MoveTo(t.endXPos+5, center-7)
		dc.ShowText(channel.name)
		dc.MoveTo(t.endXPos+5, center+7)
		dc.ShowText(channel.summaryValue(last))

		dc.Stroke()
	}
}
