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
)

type Type int

const (
	Info Type = iota
	Timing
	RaceControlMessages
	TrackMap
	Weather
	TeamRadio
	WebTiming
	RacePosition
	GapperPlot
	Telemetry
	Catching
	QualifyingImproving
)

func (t Type) String() string {
	return [...]string{
		"Info",
		"Timing",
		"RaceControlMessages",
		"TrackMap",
		"Weather",
		"TeamRadio",
		"WebTiming",
		"RacePosition",
		"GapperPlot",
		"Telemetry",
		"Catching",
		"QualifyingImproving"}[t]
}

type Panel interface {
	Type() Type

	Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig)
	Close()

	Draw(width int, height int) []giu.Widget

	ProcessDrivers(data Messages.Drivers)
	ProcessTiming(data Messages.Timing)
	ProcessEventTime(data Messages.EventTime)
	ProcessEvent(data Messages.Event)
	ProcessRaceControlMessages(data Messages.RaceControlMessage)
	ProcessWeather(data Messages.Weather)
	ProcessRadio(data Messages.Radio)
	ProcessLocation(data Messages.Location)
	ProcessTelemetry(data Messages.Telemetry)
}
