// F1Gopher - Copyright (C) 2024 f1gopher
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
	"image/color"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"golang.org/x/image/colornames"
)

type location struct {
	x float64
	y float64
}

type timedLocation struct {
	pos       location
	timestamp time.Duration
}

type fastLapInfo struct {
	prevDistanceToStartLine float64
	prevLocation            Messages.Location
	lapStartTime            time.Time
	isRecording             bool

	markers          []timedLocation
	estimatedLapTime time.Duration
	actualLapTime    time.Duration

	lapNumber    int
	driverName   string
	driverNumber int
	driverColor  color.RGBA
	position     int
	location     Messages.CarLocation

	diffToPole         time.Duration
	diffToPersonalBest time.Duration
}

type improving struct {
	dataSrc            f1gopherlib.F1GopherLib
	startLineLocations map[string]location

	fastestDriverNum int
	fastestLap       *fastLapInfo

	driverCurrentLaps map[int]*fastLapInfo
	driverFastestLaps map[int]*fastLapInfo
	//lastDriverPos     map[int]location

	startLine        location
	lastSegmentIndex int

	sortedDrivers []*fastLapInfo
	session       Messages.EventType

	lock  sync.Mutex
	table *giu.TableWidget
}

func CreateImproving() Panel {
	return &improving{
		lock: sync.Mutex{},
		startLineLocations: map[string]location{
			"Albert Park Grand Prix Circuit":              {x: -1384, y: -1155},
			"Autodromo Enzo e Dino Ferrari":               {x: -1066, y: -4778},
			"Autódromo Hermanos Rodríguez":                {x: -3524, y: -5861},
			"Autódromo Internacional do Algarve":          {x: 0, y: 0},
			"Autodromo Internazionale del Mugello":        {x: 0, y: 0},
			"Autódromo José Carlos Pace":                  {x: -744, y: 1276},
			"Autodromo Nazionale di Monza":                {x: 745, y: -5361},
			"Bahrain International Circuit":               {x: -386, y: 1170},
			"Bahrain International Circuit - Outer Track": {x: 0, y: 0},
			"Baku City Circuit":                           {x: 1079, y: -585},
			"Circuit de Barcelona-Catalunya":              {x: 1065, y: -741},
			"Circuit de Monaco":                           {x: -5992, y: -9688},
			"Circuit de Spa-Francorchamps":                {x: -307, y: 1113},
			"Circuit Gilles Villeneuve":                   {x: 1000, y: 12351},
			"Circuit of the Americas":                     {x: -769, y: -824},
			"Circuit Paul Ricard":                         {x: -534, y: -1310},
			"Circuit Park Zandvoort":                      {x: 846, y: 4604},
			"Hungaroring":                                 {x: -1675, y: 42},
			"Istanbul Park":                               {x: 0, y: 0},
			"Jeddah Corniche Circuit":                     {x: -1673, y: 1259},
			"Losail International Circuit":                {x: -1532, y: -207},
			"Marina Bay Street Circuit":                   {x: 962, y: 271},
			"Miami International Autodrome":               {x: 2171, y: -82},
			"Nürburgring":                                 {x: 0, y: 0},
			"Red Bull Ring":                               {x: 753, y: -1303},
			"Silverstone Circuit":                         {x: -1831, y: 1109},
			"Sochi Autodrom":                              {x: 0, y: 0},
			"Suzuka Circuit":                              {x: 2232, y: -1467},
			"Yas Marina Circuit":                          {x: 875, y: 2116},
			"Las Vegas Strip Street Circuit":              {x: 2362, y: -280},
		},
	}
}

func (i *improving) ProcessEventTime(data Messages.EventTime)                    {}
func (i *improving) ProcessWeather(data Messages.Weather)                        {}
func (i *improving) ProcessRadio(data Messages.Radio)                            {}
func (i *improving) ProcessTelemetry(data Messages.Telemetry)                    {}
func (i *improving) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (i *improving) Close()                                                      {}

func (i *improving) Type() Type { return QualifyingImproving }

func (i *improving) Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig) {
	i.dataSrc = dataSrc
	i.selectTrack(dataSrc.Track())
	i.driverCurrentLaps = make(map[int]*fastLapInfo)
	i.driverFastestLaps = make(map[int]*fastLapInfo)
	// i.lastDriverPos = make(map[int]location)
	i.table = nil
	i.session = Messages.Qualifying0
	i.sortedDrivers = make([]*fastLapInfo, 0)
	i.fastestDriverNum = 0
	i.fastestLap = nil
	i.lastSegmentIndex = 0

}

func (i *improving) selectTrack(name string) {
	loc, exists := i.startLineLocations[name]
	if exists {
		i.startLine = loc
		return
	}

	// Default
	i.startLine = location{}
}

func (i *improving) ProcessEvent(data Messages.Event) {
	//i.lastSegmentIndex = (data.Sector1Segments + data.Sector2Segments + data.Sector3Segments) - 1

	// Reset times when the session changes
	if data.Type != i.session {
		i.session = data.Type

		// Reset driver info's
		for _, driverInfo := range i.driverCurrentLaps {
			driverInfo.diffToPole = 0
			driverInfo.diffToPersonalBest = 0
			driverInfo.markers = []timedLocation{}
			driverInfo.prevDistanceToStartLine = math.MaxFloat64
			driverInfo.isRecording = false
		}

		i.fastestLap = nil
		i.updateTable()
	}
}

func (i *improving) ProcessDrivers(data Messages.Drivers) {

	i.sortedDrivers = []*fastLapInfo{}

	for x := range data.Drivers {
		info := &fastLapInfo{
			driverName:              data.Drivers[x].ShortName,
			driverNumber:            data.Drivers[x].Number,
			driverColor:             data.Drivers[x].Color,
			position:                data.Drivers[x].StartPosition,
			isRecording:             false,
			prevDistanceToStartLine: math.MaxFloat64,
		}

		i.driverCurrentLaps[data.Drivers[x].Number] = info
		i.sortedDrivers = append(i.sortedDrivers, info)
		//	i.lastDriverPos[data.Drivers[x].Number] = location{}
	}

	sort.Slice(
		i.sortedDrivers,
		func(x int, y int) bool {
			return i.sortedDrivers[x].position < i.sortedDrivers[y].position
		})
}

func (i *improving) ProcessTiming(data Messages.Timing) {
	if len(i.driverCurrentLaps) == 0 {
		return
	}

	// if i.startLine.x == 0 && i.startLine.y == 0 && i.lastSegmentIndex != -1 {
	// 	segment := data.Segment[i.lastSegmentIndex]
	// 	if data.Location == Messages.OnTrack || data.Location == Messages.OutLap &&
	// 		segment == Messages.InvalidSegment && segment != Messages.PitlaneSegment {
	// 		i.startLine = i.lastDriverPos[data.Number]
	// 	} else {
	// 		return
	// 	}
	// } else {
	// 	return
	// }

	driverInfo := i.driverCurrentLaps[data.Number]

	driverInfo.location = data.Location
	if driverInfo.position != data.Position {
		driverInfo.position = data.Position
		sort.Slice(
			i.sortedDrivers,
			func(x int, y int) bool {
				return i.sortedDrivers[x].position < i.sortedDrivers[y].position
			})
		i.updateTable()
	}
}

func (i *improving) ProcessLocation(data Messages.Location) {
	if data.DriverNumber == 0 {
		return
	}

	// if i.startLine.x == 0 && i.startLine.y == 0 {
	// 	i.lastDriverPos[data.DriverNumber] = location{x: data.X, y: data.Y}
	// 	return
	// }

	driverInfo := i.driverCurrentLaps[data.DriverNumber]

	// Only need to update if the driver is on a fast lap or outlap
	if driverInfo.location != Messages.OutLap && driverInfo.location != Messages.OnTrack {
		// Clear any existing info when not on track
		if driverInfo.diffToPole != 0 {
			driverInfo.diffToPole = 0
			driverInfo.diffToPersonalBest = 0
			driverInfo.markers = []timedLocation{}
			driverInfo.prevDistanceToStartLine = math.MaxFloat64
			driverInfo.isRecording = false
			i.updateTable()
		}

		return
	}

	pos := location{x: data.X, y: data.Y}
	distToStart := distance(i.startLine, pos)
	update := false

	// If the distance to start is above a threshold then do nothing
	if distToStart < 400 {
		// If not recording then check if need to start
		if !driverInfo.isRecording {
			// If getting closer to target
			if distToStart < driverInfo.prevDistanceToStartLine {
				// We don't know if we are there yet but are gettign closer
				// so do nothing yet
				driverInfo.prevDistanceToStartLine = distToStart
				driverInfo.prevLocation = data
				return
			} else {
				// We have gone past the target so use the data from the
				// previous location
				firstLoc := timedLocation{
					pos:       location{x: driverInfo.prevLocation.X, y: driverInfo.prevLocation.Y},
					timestamp: data.Timestamp.Sub(driverInfo.prevLocation.Timestamp),
				}

				driverInfo.markers = append(driverInfo.markers, firstLoc)
				driverInfo.lapStartTime = timeBetweenTwoPoints(i.startLine, driverInfo.prevLocation, data)
				driverInfo.prevLocation = data
				driverInfo.isRecording = true
				driverInfo.prevDistanceToStartLine = math.MaxFloat64
				// will store ther current point further down
			}
		} else {
			// we are recording so check if we need to stop

			// If we are past the start then ignore distance changes so we don't end the lap early
			if len(driverInfo.markers) < 20 {
				return
			} else if distToStart < driverInfo.prevDistanceToStartLine {
				// If getting closer to target

				// We don't know if we are there yet but are gettign closer
				// so do nothing yet
				driverInfo.prevDistanceToStartLine = distToStart
				driverInfo.prevLocation = data
			} else {
				// We have gone past the target so stop recording
				driverInfo.prevDistanceToStartLine = distToStart
				// Fastest lap time from the prev location
				//
				// TODO - do some average/fiddling based on how far past the start line we are?

				endTime := timeBetweenTwoPoints(i.startLine, driverInfo.prevLocation, data)
				driverInfo.estimatedLapTime = endTime.Sub(driverInfo.lapStartTime)
				driverInfo.isRecording = false

				// Store the overall fastest lap info
				if i.fastestLap == nil || i.fastestLap.estimatedLapTime > driverInfo.estimatedLapTime {
					// Needs to be a copy
					tmp := *driverInfo
					i.fastestLap = &tmp
					update = true
				}

				// Update the drivers personal best lap info
				driverFastestLap := i.driverFastestLaps[driverInfo.driverNumber]
				if driverFastestLap == nil || driverFastestLap.estimatedLapTime > driverInfo.estimatedLapTime {
					// Needs to be a copy
					tmp := *driverInfo
					i.driverFastestLaps[driverInfo.driverNumber] = &tmp
					update = true
				}

				// Reset driver tracking
				driverInfo.markers = []timedLocation{}
				driverInfo.lapStartTime = timeBetweenTwoPoints(i.startLine, driverInfo.prevLocation, data)
				driverInfo.prevDistanceToStartLine = math.MaxFloat64
				driverInfo.prevLocation = data
				driverInfo.isRecording = true

				return
			}
		}
	}

	if driverInfo.isRecording {
		// store the current point
		driverInfo.markers = append(driverInfo.markers, timedLocation{
			pos:       pos,
			timestamp: data.Timestamp.Sub(driverInfo.lapStartTime),
		})

		// Update the diff to current fastest
		if i.fastestLap != nil {
			driverInfo.diffToPole = i.diffToLap(pos, driverInfo, i.fastestLap)
			update = true
		}

		driverFastest := i.driverFastestLaps[driverInfo.driverNumber]
		if driverFastest != nil {
			driverInfo.diffToPersonalBest = i.diffToLap(pos, driverInfo, driverFastest)
			update = true
		}
	}

	if update {
		i.updateTable()
	}
}

func (i *improving) diffToLap(pos location, driverInfo *fastLapInfo, benchmark *fastLapInfo) time.Duration {

	currentMarker := driverInfo.markers[len(driverInfo.markers)-1]
	smallestDistance := math.MaxFloat64
	var smallestIndex int
	// Find the fastest lap point before to the most recent point
	for x := 0; x < len(benchmark.markers); x++ {

		currentDistance := distance(benchmark.markers[x].pos, currentMarker.pos)
		if currentDistance < smallestDistance {
			smallestDistance = currentDistance
			smallestIndex = x
		}
	}

	// TODO - do index range checks
	if smallestIndex == 0 || smallestIndex == len(benchmark.markers)-1 {
		return 0
	}

	// Find the second closest point (is it before or after?e
	beforeDist := distance(benchmark.markers[smallestIndex-1].pos, currentMarker.pos)
	afterDist := distance(benchmark.markers[smallestIndex+1].pos, currentMarker.pos)

	// Current is between before and smallest
	var start, end timedLocation
	if beforeDist < afterDist {
		start = benchmark.markers[smallestIndex-1]
		end = benchmark.markers[smallestIndex]
	} else {
		start = benchmark.markers[smallestIndex]
		end = benchmark.markers[smallestIndex+1]
	}

	// Normalize the recent point back to the fastest point
	timeDiff := timeBetweenTwoPoints(pos,
		Messages.Location{X: start.pos.x, Y: start.pos.y, Timestamp: benchmark.lapStartTime.Add(start.timestamp)},
		Messages.Location{X: end.pos.x, Y: end.pos.y, Timestamp: benchmark.lapStartTime.Add(end.timestamp)})
	fastestElapsed := timeDiff.Sub(benchmark.lapStartTime)

	// Update time diff
	elapsedLapTime := currentMarker.timestamp
	return elapsedLapTime - fastestElapsed
}

func (i *improving) updateTable() {
	if i.fastestLap == nil {
		return
	}

	rows := []*giu.TableRowWidget{}

	for _, driver := range i.sortedDrivers {
		parts := []giu.Widget{
			giu.Labelf("%d", driver.position),
			giu.Style().SetColor(giu.StyleColorText, driver.driverColor).To(giu.Label(driver.driverName)),
		}

		if driver.location == Messages.OnTrack {
			// Show comparison to personal best if the driver has a stored fast lap
			if _, exists := i.driverFastestLaps[driver.driverNumber]; exists {
				timeColor := colornames.White
				if driver.diffToPersonalBest > 0 {
					timeColor = colornames.Red
				} else if driver.diffToPersonalBest < 0 {
					timeColor = colornames.Green
				}

				parts = append(parts,
					giu.Style().SetColor(
						giu.StyleColorText, timeColor).
						To(giu.Label(fmtDurationNoMins(driver.diffToPersonalBest))))
			} else {
				parts = append(parts, giu.Label("      -"))
			}

			timeColor := colornames.White
			if driver.diffToPole > 0 {
				timeColor = colornames.Red
			} else if driver.diffToPole < 0 {
				timeColor = colornames.Green
			}

			parts = append(parts,
				giu.Style().SetColor(
					giu.StyleColorText, timeColor).
					To(giu.Label(fmtDurationNoMins(driver.diffToPole))))

		} else {
			parts = append(parts, giu.Label("      -"))
			parts = append(parts, giu.Label("      -"))
		}

		rows = append(rows, giu.TableRow(parts...))
	}

	result := giu.Table().Flags(giu.TableFlagsResizable|giu.TableFlagsSizingFixedSame).
		Columns(
			giu.TableColumn("Pos").InnerWidthOrWeight(50),
			giu.TableColumn("Driver").InnerWidthOrWeight(70),
			giu.TableColumn("Personal Δ").InnerWidthOrWeight(100),
			giu.TableColumn("Pole Δ").InnerWidthOrWeight(100),
		).Rows(rows...)

	i.lock.Lock()
	i.table = result
	i.lock.Unlock()
}

func (i *improving) Draw(width int, height int) []giu.Widget {
	// If it's the first session and we have no fast lap then display nothing
	// any session after the frist won't have a fastest but will have personal
	// laps from the first session so we can display something
	if i.fastestLap == nil && i.session == Messages.Qualifying1 || i.table == nil {
		return []giu.Widget{
			giu.Label("Waiting for the first fast lap..."),
		}
	}

	results := []giu.Widget{}
	i.lock.Lock()
	results = append(results, i.table)
	i.lock.Unlock()
	if i.fastestLap != nil {
		results = append(results, giu.Labelf("Fastest: %s %v", i.fastestLap.driverName, fmtDuration(i.fastestLap.estimatedLapTime)))
	}

	return results
}

func distance(a location, b location) float64 {
	x := math.Pow(b.x-a.x, 2)
	y := math.Pow(b.y-a.y, 2)
	return math.Sqrt(x + y)
}

func timeBetweenTwoPoints(target location, a Messages.Location, b Messages.Location) time.Time {
	earliest := a
	latest := b
	if b.Timestamp.Before(a.Timestamp) {
		earliest = b
		latest = a
	}

	beforeDist := distance(target, location{x: earliest.X, y: earliest.Y})
	afterDist := distance(target, location{x: latest.X, y: latest.Y})

	if beforeDist == 0 {
		return earliest.Timestamp
	} else if afterDist == 0 {
		return latest.Timestamp
	}

	distBetweenAB := afterDist + beforeDist

	timeDiff := latest.Timestamp.UnixMilli() - earliest.Timestamp.UnixMilli()

	// Percent between 1.0 and 0.0
	percentDist := beforeDist / distBetweenAB

	resultMilli := float64(timeDiff) * percentDist

	abc := earliest.Timestamp.Add(time.Millisecond * time.Duration(int64(resultMilli)))

	return abc
}
