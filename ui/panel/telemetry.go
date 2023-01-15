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
	"image/color"
	"sort"
	"sync/atomic"
	"time"
)

type floatPair struct {
	time  time.Time
	value float32
}

type boolPair struct {
	time  time.Time
	value bool
}

type intPair struct {
	time  time.Time
	value int16
}

type bytePair struct {
	time  time.Time
	value byte
}

type telemetryInfo struct {
	name    string
	color   color.RGBA
	number  int
	enabled bool
	display bool

	rpm      []intPair
	speed    []floatPair
	gear     []bytePair
	throttle []floatPair
	brake    []floatPair
	drs      []boolPair
}

type telemetry struct {
	data          map[int]*telemetryInfo
	currentDriver int
	refresh       atomic.Bool
	driverNames   []string

	dataSelect   *driverDataSelectWidget
	driverSelect *driverDisplaySelectWidget

	lines []giu.PlotWidget
}

func CreateTelemetry() Panel {
	return &telemetry{
		data:  map[int]*telemetryInfo{},
		lines: []giu.PlotWidget{},
		dataSelect: &driverDataSelectWidget{
			id: "telemetryDataSelect",
		},
		driverSelect: &driverDisplaySelectWidget{
			id: "telemetryDriverSelect",
		},
	}
}

func (t *telemetry) ProcessTiming(data Messages.Timing)                          {}
func (t *telemetry) ProcessEventTime(data Messages.EventTime)                    {}
func (t *telemetry) ProcessEvent(data Messages.Event)                            {}
func (t *telemetry) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (t *telemetry) ProcessWeather(data Messages.Weather)                        {}
func (t *telemetry) ProcessRadio(data Messages.Radio)                            {}
func (t *telemetry) ProcessLocation(data Messages.Location)                      {}
func (t *telemetry) Close()                                                      {}

func (t *telemetry) Type() Type { return Telemetry }

func (t *telemetry) Init(dataSrc f1gopherlib.F1GopherLib) {
	t.data = map[int]*telemetryInfo{}
	t.lines = []giu.PlotWidget{}
	t.currentDriver = -1
	t.refresh.Store(false)
	t.dataSelect.drivers = nil
	t.driverNames = nil
}

func (t *telemetry) ProcessDrivers(data Messages.Drivers) {
	for x := range data.Drivers {
		driver := &telemetryInfo{
			name:     data.Drivers[x].ShortName,
			number:   data.Drivers[x].Number,
			color:    data.Drivers[x].Color,
			enabled:  false,
			rpm:      []intPair{},
			speed:    []floatPair{},
			gear:     []bytePair{},
			throttle: []floatPair{},
			brake:    []floatPair{},
			drs:      []boolPair{},
		}
		t.data[data.Drivers[x].Number] = driver
		t.dataSelect.drivers = append(t.dataSelect.drivers, driver)
		t.driverSelect.drivers = append(t.driverSelect.drivers, driver)
		t.driverNames = append(t.driverNames, driver.name)
	}

	sort.Slice(t.dataSelect.drivers, func(i, j int) bool {
		return t.dataSelect.drivers[i].name < t.dataSelect.drivers[j].name
	})
	sort.Slice(t.driverSelect.drivers, func(i, j int) bool {
		return t.driverSelect.drivers[i].name < t.driverSelect.drivers[j].name
	})
	sort.Strings(t.driverNames)
}

func (t *telemetry) ProcessTelemetry(data Messages.Telemetry) {
	driverInfo, exists := t.data[data.DriverNumber]
	if !exists || !driverInfo.enabled {
		return
	}

	driverInfo.rpm = append(driverInfo.rpm, intPair{time: data.Timestamp, value: data.RPM})
	driverInfo.speed = append(driverInfo.speed, floatPair{time: data.Timestamp, value: data.Speed})
	driverInfo.gear = append(driverInfo.gear, bytePair{time: data.Timestamp, value: data.Gear})
	driverInfo.throttle = append(driverInfo.throttle, floatPair{time: data.Timestamp, value: data.Throttle})
	driverInfo.brake = append(driverInfo.brake, floatPair{time: data.Timestamp, value: data.Brake})
	driverInfo.drs = append(driverInfo.drs, boolPair{time: data.Timestamp, value: data.DRS})

	if t.driverSelect.selected != nil && data.DriverNumber == t.driverSelect.selected.number {
		t.refresh.Store(true)
	}
}

func (t *telemetry) Draw(width int, height int) []giu.Widget {
	if t.refresh.Load() {
		t.refresh.Store(false)

		d := []float64{}
		for x := range t.driverSelect.selected.speed {
			d = append(d, float64(t.driverSelect.selected.speed[x].value))
		}

		t.lines = []giu.PlotWidget{giu.PlotLine("Speed", d)}
	}

	return []giu.Widget{
		giu.Row(
			t.dataSelect,
			t.driverSelect,
		),
		giu.Plot("Telemetry").Plots(t.lines...).
			Size(width-16, height-36),
	}
}

type driverDataSelectWidget struct {
	id         string
	drivers    []*telemetryInfo
	numEnabled int
}

func (c *driverDataSelectWidget) Build() {
	if imgui.BeginCombo("Store Data For", fmt.Sprintf("%d drivers", c.numEnabled)) {

		for x := range c.drivers {
			imgui.Checkbox(c.drivers[x].name, &c.drivers[x].enabled)
		}

		imgui.EndCombo()

		c.numEnabled = 0
		for x := range c.drivers {
			if c.drivers[x].enabled {
				c.numEnabled++
			}
		}
	}
}

type driverDisplaySelectWidget struct {
	id       string
	drivers  []*telemetryInfo
	selected *telemetryInfo
}

func (c *driverDisplaySelectWidget) Build() {

	selected := ""
	if c.selected != nil {
		selected = c.selected.name
	}

	if imgui.BeginCombo("Display Data For", selected) {

		for x := range c.drivers {
			if imgui.Checkbox(c.drivers[x].name, &c.drivers[x].display) {
				c.selected = c.drivers[x]
			}
		}

		imgui.EndCombo()
	}
}
