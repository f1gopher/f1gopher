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
	"golang.org/x/image/colornames"
	"image/color"
	"sort"
	"sync"
	"time"
)

type timing struct {
	data     map[int]Messages.Timing
	dataLock sync.Mutex

	event     Messages.Event
	eventLock sync.Mutex

	fastestSector1        time.Duration
	fastestSector2        time.Duration
	fastestSector3        time.Duration
	theoreticalFastestLap time.Duration
	previousSessionActive Messages.SessionState
	fastestSpeedTrap      int
	timeLostInPitlane     time.Duration

	gapToInfront         bool
	isRaceSession        bool
	predictedPitstopTime time.Duration

	table *giu.TableWidget
}

const timeWidth = 75

func CreateTiming() Panel {
	return &timing{
		data: make(map[int]Messages.Timing),
	}
}

func (t *timing) ProcessDrivers(data Messages.Drivers)                        {}
func (t *timing) ProcessEventTime(data Messages.EventTime)                    {}
func (t *timing) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (t *timing) ProcessWeather(data Messages.Weather)                        {}
func (t *timing) ProcessRadio(data Messages.Radio)                            {}
func (t *timing) ProcessLocation(data Messages.Location)                      {}
func (t *timing) ProcessTelemetry(data Messages.Telemetry)                    {}
func (t *timing) Close()                                                      {}

func (t *timing) Type() Type { return Timing }

func (t *timing) Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig) {
	t.gapToInfront = dataSrc.Session() == Messages.RaceSession || dataSrc.Session() == Messages.SprintSession

	// Clear any previous session data
	t.data = make(map[int]Messages.Timing)
	t.fastestSector1 = 0
	t.fastestSector2 = 0
	t.fastestSector3 = 0
	t.theoreticalFastestLap = 0
	t.previousSessionActive = Messages.UnknownState
	t.fastestSpeedTrap = 0
	t.timeLostInPitlane = dataSrc.TimeLostInPitlane()
	t.isRaceSession = dataSrc.Session() == Messages.RaceSession || dataSrc.Session() == Messages.SprintSession
	t.predictedPitstopTime = config.PredictedPitstopTime()

	t.table = giu.Table().FastMode(true).Flags(giu.TableFlagsResizable | giu.TableFlagsSizingFixedSame)
	columns := []*giu.TableColumnWidget{
		giu.TableColumn("Pos").InnerWidthOrWeight(25),
		giu.TableColumn("Driver").InnerWidthOrWeight(40),
		giu.TableColumn("Segment").InnerWidthOrWeight(200),
		giu.TableColumn("Fastest").InnerWidthOrWeight(timeWidth),
		giu.TableColumn("Gap").InnerWidthOrWeight(timeWidth),
		giu.TableColumn("S1").InnerWidthOrWeight(timeWidth),
		giu.TableColumn("S2").InnerWidthOrWeight(timeWidth),
		giu.TableColumn("S3").InnerWidthOrWeight(timeWidth),
		giu.TableColumn("Last Lap").InnerWidthOrWeight(timeWidth),
		giu.TableColumn("DRS").InnerWidthOrWeight(50),
		giu.TableColumn("Tire").InnerWidthOrWeight(50),
		giu.TableColumn("Lap").InnerWidthOrWeight(30),
	}

	if t.isRaceSession {
		columns = append(columns, []*giu.TableColumnWidget{
			giu.TableColumn("Pitstops").InnerWidthOrWeight(60),
			giu.TableColumn("Pit Time").InnerWidthOrWeight(timeWidth),
			giu.TableColumn("Post Pit Pos").InnerWidthOrWeight(100),
		}...)
	}

	columns = append(columns, []*giu.TableColumnWidget{
		giu.TableColumn("Spd Trp").InnerWidthOrWeight(50),
		giu.TableColumn("Location").InnerWidthOrWeight(70),
	}...)

	t.table.Columns(columns...)
}

func (t *timing) ProcessTiming(data Messages.Timing) {
	t.dataLock.Lock()
	t.data[data.Number] = data
	t.dataLock.Unlock()
}

func (t *timing) ProcessEvent(data Messages.Event) {
	t.eventLock.Lock()
	t.event = data
	t.eventLock.Unlock()
}

func (t *timing) Draw(width int, height int) []giu.Widget {

	drivers := t.orderedDrivers()

	t.updateSessionStats(drivers)

	t.eventLock.Lock()
	totalSegments := t.event.TotalSegments
	sector1Segments := t.event.Sector1Segments
	sector2Segments := t.event.Sector2Segments
	t.eventLock.Unlock()

	// Driver rows
	var rows []*giu.TableRowWidget
	for x := range drivers {
		// DRS
		drs := "Closed"
		if drivers[x].DRSOpen {
			drs = "Open"
		}
		drsColor := colornames.White
		// TODO - only green when track DRS state is enabled or unknown
		if drivers[x].TimeDiffToPositionAhead > 0 && drivers[x].TimeDiffToPositionAhead < time.Second {
			drsColor = colornames.Green
		}

		// Speed Trap
		speedTrap := ""
		if drivers[x].SpeedTrap > 0 {
			speedTrap = fmt.Sprintf("%d", drivers[x].SpeedTrap)
		}

		// Calculate driver segments
		segments := []giu.Widget{}
		for s := 0; s < totalSegments; s++ {
			switch drivers[x].Segment[s] {
			case Messages.None:
				segments = append(segments, giu.Style().SetColor(giu.StyleColorText, colornames.Lightgray).To(
					giu.Label("■")))
			default:
				segments = append(segments, giu.Style().SetColor(giu.StyleColorText, segmentColor(drivers[x].Segment[s])).To(
					giu.Label("■")))
			}

			if s == sector1Segments-1 || s == sector1Segments+sector2Segments-1 {
				segments = append(segments, giu.Label("|"))
			}
		}

		// Gap
		gap := drivers[x].TimeDiffToFastest
		if t.gapToInfront {
			gap = drivers[x].TimeDiffToPositionAhead
		}

		lastPitlaneTime := ""
		if len(drivers[x].PitStopTimes) > 0 {
			lastPitlane := &drivers[x].PitStopTimes[len(drivers[x].PitStopTimes)-1]

			if lastPitlane.PitlaneTime != 0 {
				lastPitlaneTime = fmtDuration(lastPitlane.PitlaneTime)
			}
		}

		positionsLost := drivers[x].Position
		positionColor := colornames.Green
		pitTimeLost := t.timeLostInPitlane + t.predictedPitstopTime
		for driverBehind := x + 1; driverBehind < len(drivers); driverBehind++ {
			pitTimeLost = pitTimeLost - drivers[driverBehind].TimeDiffToPositionAhead

			if pitTimeLost <= 0 {
				break
			}

			// Can't drop below stopped cars
			if drivers[driverBehind].Location == Messages.Stopped ||
				drivers[driverBehind].Location == Messages.OutOfRace {
				break
			}

			positionsLost++
		}

		potentialPositionChange := fmt.Sprintf("%d", positionsLost)
		if positionsLost != drivers[x].Position {
			positionColor = colornames.Red
		}

		widgets := []giu.Widget{
			giu.Label(fmt.Sprintf("%d", drivers[x].Position)),
			giu.Style().SetColor(giu.StyleColorText, drivers[x].Color).To(
				giu.Label(drivers[x].ShortName)),

			giu.Style().SetStyleFloat(giu.StyleVarItemSpacing, 0).To(giu.Row(segments...)),

			giu.Style().SetColor(giu.StyleColorText, fastestLapColor(drivers[x].OverallFastestLap)).To(
				giu.Label(fmtDuration(drivers[x].FastestLap))),
			giu.Label(fmtDuration(gap)),
			giu.Style().SetColor(giu.StyleColorText, timeColor(drivers[x].Sector1PersonalFastest, drivers[x].Sector1OverallFastest)).To(
				giu.Label(fmtDuration(drivers[x].Sector1))),
			giu.Style().SetColor(giu.StyleColorText, timeColor(drivers[x].Sector2PersonalFastest, drivers[x].Sector2OverallFastest)).To(
				giu.Label(fmtDuration(drivers[x].Sector2))),
			giu.Style().SetColor(giu.StyleColorText, timeColor(drivers[x].Sector3PersonalFastest, drivers[x].Sector3OverallFastest)).To(
				giu.Label(fmtDuration(drivers[x].Sector3))),
			giu.Style().SetColor(giu.StyleColorText, timeColor(drivers[x].LastLapPersonalFastest, drivers[x].LastLapOverallFastest)).To(
				giu.Label(fmtDuration(drivers[x].LastLap))),
			giu.Style().SetColor(giu.StyleColorText, drsColor).To(
				giu.Label(drs)),
			giu.Style().SetColor(giu.StyleColorText, tireColor(drivers[x].Tire)).To(
				giu.Label(drivers[x].Tire.String())),
			giu.Label(fmt.Sprintf("%d", drivers[x].LapsOnTire)),
		}

		if t.isRaceSession {
			widgets = append(widgets, []giu.Widget{
				giu.Label(fmt.Sprintf("%d", drivers[x].Pitstops)),
				giu.Label(lastPitlaneTime),
				giu.Style().SetColor(giu.StyleColorText, positionColor).To(
					giu.Label(potentialPositionChange)),
			}...)
		}

		widgets = append(widgets, []giu.Widget{
			giu.Style().SetColor(giu.StyleColorText, timeColor(drivers[x].SpeedTrapPersonalFastest, drivers[x].SpeedTrapOverallFastest)).To(
				giu.Label(speedTrap)),
			giu.Style().SetColor(giu.StyleColorText, locationColor(drivers[x].Location)).To(
				giu.Label(drivers[x].Location.String())),
		}...)

		rows = append(rows, giu.TableRow(widgets...))
	}

	// Track segments
	trackSegments := []giu.Widget{}
	for s := 0; s < totalSegments; s++ {
		switch t.event.SegmentFlags[s] {
		case Messages.GreenFlag:
			trackSegments = append(trackSegments, giu.Style().SetColor(giu.StyleColorText, colornames.Green).To(
				giu.Label("■")))

		case Messages.YellowFlag:
			trackSegments = append(trackSegments, giu.Style().SetColor(giu.StyleColorText, colornames.Yellow).To(
				giu.Label("■")))

		case Messages.DoubleYellowFlag:
			trackSegments = append(trackSegments, giu.Style().SetColor(giu.StyleColorText, color.RGBA{
				R: 251,
				G: 255,
				B: 0,
				A: 0xFF,
			}).To(giu.Label("■")))

		case Messages.RedFlag:
			trackSegments = append(trackSegments, giu.Style().SetColor(giu.StyleColorText, colornames.Red).To(
				giu.Label("■")))
		}

		if s == sector1Segments-1 || s == sector1Segments+sector2Segments-1 {
			trackSegments = append(trackSegments, giu.Label("|"))
		}
	}

	// Session/track info row
	rowWidgets := []giu.Widget{
		giu.Label(""),
		giu.Label("Track:"),
		giu.Style().SetStyleFloat(giu.StyleVarItemSpacing, 0).To(giu.Row(trackSegments...)),
		giu.Label(""),
		giu.Label("Session:"),
		giu.Style().SetColor(giu.StyleColorText, colornames.Purple).To(giu.Label(fmtDuration(t.fastestSector1))),
		giu.Style().SetColor(giu.StyleColorText, colornames.Purple).To(giu.Label(fmtDuration(t.fastestSector2))),
		giu.Style().SetColor(giu.StyleColorText, colornames.Purple).To(giu.Label(fmtDuration(t.fastestSector3))),
		giu.Style().SetColor(giu.StyleColorText, colornames.Purple).To(giu.Label(fmtDuration(t.theoreticalFastestLap))),
		giu.Label(""),
		giu.Label(""),
		giu.Label(""),
	}

	if t.isRaceSession {
		rowWidgets = append(rowWidgets, []giu.Widget{
			giu.Style().SetColor(giu.StyleColorText, colornames.Purple).To(giu.Label(fmt.Sprintf("%d", t.fastestSpeedTrap))),
			giu.Label(""),
			giu.Label(""),
			giu.Label(""),
		}...)
	} else {
		rowWidgets = append(rowWidgets, []giu.Widget{
			giu.Label(""),
			giu.Label(""),
		}...)
	}

	rows = append(rows, giu.TableRow(rowWidgets...))

	return []giu.Widget{t.table.Rows(rows...)}
}

func (t *timing) updateSessionStats(drivers []Messages.Timing) {

	// Track the fastest sectors times for the session
	for _, driver := range drivers {
		if (driver.Sector1 > 0 && driver.Sector1 < t.fastestSector1) || t.fastestSector1 == 0 {
			t.fastestSector1 = driver.Sector1
		}

		if (driver.Sector2 > 0 && driver.Sector2 < t.fastestSector2) || t.fastestSector2 == 0 {
			t.fastestSector2 = driver.Sector2
		}

		if (driver.Sector3 > 0 && driver.Sector3 < t.fastestSector3) || t.fastestSector3 == 0 {
			t.fastestSector3 = driver.Sector3
		}

		if driver.SpeedTrap > t.fastestSpeedTrap {
			t.fastestSpeedTrap = driver.SpeedTrap
		}
	}

	if t.fastestSector1 > 0 && t.fastestSector2 > 0 && t.fastestSector3 > 0 {
		t.theoreticalFastestLap = t.fastestSector1 + t.fastestSector2 + t.fastestSector3
	}

	if t.event.Status == Messages.Started {
		if t.previousSessionActive != Messages.Started {
			t.fastestSector1 = 0
			t.fastestSector2 = 0
			t.fastestSector3 = 0
			t.theoreticalFastestLap = 0
			t.previousSessionActive = t.event.Status
		}
	} else if t.event.Status == Messages.Inactive {
		t.fastestSector1 = 0
		t.fastestSector2 = 0
		t.fastestSector3 = 0
		t.theoreticalFastestLap = 0
		t.previousSessionActive = t.event.Status
	} else {
		t.previousSessionActive = t.event.Status
	}
}

func (t *timing) orderedDrivers() []Messages.Timing {
	drivers := make([]Messages.Timing, 0)
	t.dataLock.Lock()
	for _, a := range t.data {
		drivers = append(drivers, a)
	}
	t.dataLock.Unlock()

	// Sort drivers into their position order
	sort.Slice(drivers, func(i, j int) bool {
		return drivers[i].Position < drivers[j].Position
	})
	return drivers
}
