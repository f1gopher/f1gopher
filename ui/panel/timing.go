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
	"image/color"
	"sort"
	"sync"
	"time"

	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"golang.org/x/image/colornames"
)

type timing struct {
	data     map[int]Messages.Timing
	dataLock sync.Mutex

	event     Messages.Event
	eventLock sync.Mutex

	fastestSector1         time.Duration
	fastestSector1Driver   string
	fastestSector2         time.Duration
	fastestSector2Driver   string
	fastestSector3         time.Duration
	fastestSector3Driver   string
	theoreticalFastestLap  time.Duration
	fastestSpeedTrap       int
	fastestSpeedTrapDriver string
	timeLostInPitlane      time.Duration
	previousEventType      Messages.EventType

	gapToInfront        bool
	isRaceSession       bool
	isSprintRaceSession bool
	config              PanelConfig

	lastPitLossColor map[int]color.RGBA
	lastPitLossValue map[int]string

	table *giu.TableWidget
}

const timeWidth = 75

var outBackground = color.RGBA{R: 69, G: 69, B: 228, A: 255}
var dropZoneBackground = color.RGBA{R: 83, G: 84, B: 78, A: 255}
var defaultBackgroundColor = color.RGBA{R: 0, G: 0, B: 0, A: 0}
var altDefaultBackgroundColor = color.RGBA{R: 55, G: 55, B: 55, A: 255}

func CreateTiming() Panel {
	return &timing{
		data:             make(map[int]Messages.Timing),
		lastPitLossColor: make(map[int]color.RGBA),
		lastPitLossValue: make(map[int]string),
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
	t.lastPitLossColor = make(map[int]color.RGBA)
	t.lastPitLossValue = make(map[int]string)
	t.fastestSector1 = 0
	t.fastestSector1Driver = ""
	t.fastestSector2 = 0
	t.fastestSector2Driver = ""
	t.fastestSector3 = 0
	t.fastestSector3Driver = ""
	t.theoreticalFastestLap = 0
	t.fastestSpeedTrapDriver = ""
	t.previousEventType = Messages.PreSeason
	t.fastestSpeedTrap = 0
	t.timeLostInPitlane = dataSrc.TimeLostInPitlane()
	t.isRaceSession = dataSrc.Session() == Messages.RaceSession
	t.isSprintRaceSession = dataSrc.Session() == Messages.SprintSession
	t.config = config

	t.table = giu.Table().FastMode(true).Flags(giu.TableFlagsResizable | giu.TableFlagsSizingFixedSame)
	columns := []*giu.TableColumnWidget{
		giu.TableColumn("Pos").InnerWidthOrWeight(25),
		giu.TableColumn("Driver").InnerWidthOrWeight(41),
		giu.TableColumn("Segment").InnerWidthOrWeight(240),
		giu.TableColumn("Fastest").InnerWidthOrWeight(timeWidth),
		giu.TableColumn("Gap").InnerWidthOrWeight(60),
		giu.TableColumn("S1").InnerWidthOrWeight(60),
		giu.TableColumn("S2").InnerWidthOrWeight(60),
		giu.TableColumn("S3").InnerWidthOrWeight(60),
		giu.TableColumn("Last Lap").InnerWidthOrWeight(timeWidth),
		giu.TableColumn("DRS").InnerWidthOrWeight(50),
		giu.TableColumn("Tire").InnerWidthOrWeight(50),
		giu.TableColumn("Lap").InnerWidthOrWeight(30),
	}

	if t.isRaceSession {
		columns = append(columns, []*giu.TableColumnWidget{
			giu.TableColumn("Pits").InnerWidthOrWeight(30),
			giu.TableColumn("Pit Time").InnerWidthOrWeight(50),
			giu.TableColumn("Post Pit Pos").InnerWidthOrWeight(100),
		}...)
	} else if t.isSprintRaceSession {
		columns = append(columns, []*giu.TableColumnWidget{
			giu.TableColumn("Pitstops").InnerWidthOrWeight(60),
			giu.TableColumn("Pit Time").InnerWidthOrWeight(timeWidth),
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
	predictedPitstopTime := t.config.PredictedPitstopTime()

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
		// Can use DRS if a race session and close enough to the car infront or if not a race session
		drsColor := colornames.White
		if t.event.DRSEnabled != Messages.DRSDisabled &&
			((t.event.Type != Messages.Race && t.event.Type != Messages.Sprint) ||
				(drivers[x].TimeDiffToPositionAhead > 0 && drivers[x].TimeDiffToPositionAhead < time.Second)) {
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
				segments = append(segments, giu.Style().SetColor(giu.StyleColorText, color.RGBA{R: 230, G: 230, B: 230, A: 80}).To(
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
			var potentialPositionChange string
			positionColor := colornames.Green

			if drivers[x].Location != Messages.Pitlane && drivers[x].Location != Messages.PitOut {
				newPosition := drivers[x].Position
				pitTimeLost := t.timeLostInPitlane + predictedPitstopTime
				// Default value for the last car because we won't enter the loop
				var timeToCarAhead = pitTimeLost
				var timeToCarBehind time.Duration

				timeToClearPitlane := time.Second * 10
				gapToCar := time.Second * 0
				minGap := (pitTimeLost - timeToClearPitlane)

				for driverBehind := x + 1; driverBehind < len(drivers); driverBehind++ {
					//timeToCarAhead = pitTimeLost

					// Can't drop below stopped cars
					if drivers[driverBehind].Location == Messages.Stopped ||
						drivers[driverBehind].Location == Messages.OutOfRace {
						break
					}

					newPosition++

					gapToCar += drivers[driverBehind].TimeDiffToPositionAhead

					// If the gap to the prediction car is less than the time to drive past the pitlane then keep looking
					if gapToCar < minGap {
						continue
					}

					timeToCarBehind = gapToCar - minGap
					timeToCarAhead = drivers[driverBehind].TimeDiffToPositionAhead - timeToCarBehind
					break
				}

				if newPosition == drivers[x].Position {
					timeToCarAhead += drivers[x].TimeDiffToPositionAhead
				}

				if drivers[x].Location != Messages.Stopped && drivers[x].Location != Messages.OutOfRace {
					if newPosition == 1 {
						potentialPositionChange = fmt.Sprintf("%02d>%s", newPosition, t.fmtGapDuration(timeToCarBehind))
					} else if timeToCarBehind == 0 {
						potentialPositionChange = fmt.Sprintf("%s<%02d", t.fmtGapDuration(timeToCarAhead), newPosition)
					} else {
						potentialPositionChange = fmt.Sprintf("%s<%02d>%s", t.fmtGapDuration(timeToCarAhead), newPosition, t.fmtGapDuration(timeToCarBehind))
					}
				}

				if newPosition != drivers[x].Position {
					positionColor = colornames.Red
				}

				t.lastPitLossColor[drivers[x].Number] = positionColor
				t.lastPitLossValue[drivers[x].Number] = potentialPositionChange
			} else {
				positionColor = t.lastPitLossColor[drivers[x].Number]
				potentialPositionChange = t.lastPitLossValue[drivers[x].Number]
			}

			widgets = append(widgets, []giu.Widget{
				giu.Label(fmt.Sprintf("%d", drivers[x].Pitstops)),
				giu.Label(lastPitlaneTime),
				giu.Style().SetColor(giu.StyleColorText, positionColor).To(
					giu.Label(potentialPositionChange)),
			}...)
		} else if t.isSprintRaceSession {
			widgets = append(widgets, []giu.Widget{
				giu.Label(fmt.Sprintf("%d", drivers[x].Pitstops)),
				giu.Label(lastPitlaneTime),
			}...)
		}

		widgets = append(widgets, []giu.Widget{
			giu.Style().SetColor(giu.StyleColorText, timeColor(drivers[x].SpeedTrapPersonalFastest, drivers[x].SpeedTrapOverallFastest)).To(
				giu.Label(speedTrap)),
			giu.Style().SetColor(giu.StyleColorText, locationColor(drivers[x].Location)).To(
				giu.Label(drivers[x].Location.String())),
		}...)

		// For qualifying show which drivers are out or in the drop zone by changing the background color
		backgroundColor := defaultBackgroundColor
		if t.event.Type == Messages.Qualifying1 {
			if drivers[x].Position > 15 {
				backgroundColor = dropZoneBackground
			}

		} else if t.event.Type == Messages.Qualifying2 {
			if drivers[x].Position > 15 {
				backgroundColor = outBackground
			} else if drivers[x].Position > 10 {
				backgroundColor = dropZoneBackground
			}

		} else if t.event.Type == Messages.Qualifying3 {
			if drivers[x].Position > 10 {
				backgroundColor = outBackground
			}
		} else {
			// When not a qualifying session alternate the row background color to make things more readable
			if drivers[x].Position%2 == 0 {
				backgroundColor = altDefaultBackgroundColor
			}
		}

		rows = append(rows, giu.TableRow(widgets...).BgColor(backgroundColor))
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
		giu.Style().SetColor(giu.StyleColorText, purpleColor).To(giu.Label(fmtDuration(t.fastestSector1))),
		giu.Tooltip(t.fastestSector1Driver),
		giu.Style().SetColor(giu.StyleColorText, purpleColor).To(giu.Label(fmtDuration(t.fastestSector2))),
		giu.Tooltip(t.fastestSector2Driver),
		giu.Style().SetColor(giu.StyleColorText, purpleColor).To(giu.Label(fmtDuration(t.fastestSector3))),
		giu.Tooltip(t.fastestSector3Driver),
		giu.Style().SetColor(giu.StyleColorText, purpleColor).To(giu.Label(fmtDuration(t.theoreticalFastestLap))),
		giu.Label(""),
		giu.Label(""),
		giu.Label(""),
	}

	if t.isRaceSession || t.isSprintRaceSession {
		rowWidgets = append(rowWidgets, []giu.Widget{
			giu.Label(""),
			giu.Label(""),
			giu.Label(""),
			giu.Style().SetColor(giu.StyleColorText, purpleColor).To(giu.Label(fmt.Sprintf("%d", t.fastestSpeedTrap))),
			giu.Tooltip(t.fastestSpeedTrapDriver),
		}...)
	} else {
		rowWidgets = append(rowWidgets,
			giu.Style().SetColor(giu.StyleColorText, purpleColor).To(giu.Label(fmt.Sprintf("%d", t.fastestSpeedTrap))),
			giu.Tooltip(t.fastestSpeedTrapDriver),
		)
	}

	rows = append(rows, giu.TableRow(rowWidgets...))

	return []giu.Widget{t.table.Rows(rows...)}
}

func (t *timing) updateSessionStats(drivers []Messages.Timing) {

	// Track the fastest sectors times for the session
	for _, driver := range drivers {
		if (driver.Sector1 > 0 && driver.Sector1 < t.fastestSector1) || t.fastestSector1 == 0 {
			t.fastestSector1 = driver.Sector1
			t.fastestSector1Driver = driver.ShortName
		}

		if (driver.Sector2 > 0 && driver.Sector2 < t.fastestSector2) || t.fastestSector2 == 0 {
			t.fastestSector2 = driver.Sector2
			t.fastestSector2Driver = driver.ShortName
		}

		if (driver.Sector3 > 0 && driver.Sector3 < t.fastestSector3) || t.fastestSector3 == 0 {
			t.fastestSector3 = driver.Sector3
			t.fastestSector3Driver = driver.ShortName
		}

		if driver.SpeedTrap > t.fastestSpeedTrap {
			t.fastestSpeedTrap = driver.SpeedTrap
			t.fastestSpeedTrapDriver = driver.ShortName
		}
	}

	if t.fastestSector1 > 0 && t.fastestSector2 > 0 && t.fastestSector3 > 0 {
		t.theoreticalFastestLap = t.fastestSector1 + t.fastestSector2 + t.fastestSector3
	}

	// If the event type changes then reset everything (will handle different quali sessions and if the app is kept
	// open all day)
	if t.previousEventType != t.event.Type {
		t.fastestSector1 = 0
		t.fastestSector2 = 0
		t.fastestSector3 = 0
		t.theoreticalFastestLap = 0
		t.previousEventType = t.event.Type
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

func (t *timing) fmtGapDuration(d time.Duration) string {
	milliseconds := d.Milliseconds()

	minutes := milliseconds / (1000 * 60)
	milliseconds -= minutes * 60 * 1000
	seconds := milliseconds / 1000
	milliseconds -= seconds * 1000
	// Only want to display to a tenth of a second
	milliseconds = milliseconds / 100

	// If no minutes then don't display zero but pad with spaces for display alignment
	if minutes == 0 {
		return fmt.Sprintf("%02d.%01d", seconds, milliseconds)
	}

	return fmt.Sprintf("%02d:%02d.%01d", minutes, seconds, milliseconds)
}
