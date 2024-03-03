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
	"math"
	"sort"
	"time"

	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
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
	lapNumber        int
	driverName       string
	driverNumber     int
	location         Messages.CarLocation

	diffToFastest time.Duration
}

type improving struct {
	dataSrc f1gopherlib.F1GopherLib

	fastestDriverNum int
	// fastestOnQualiLap    bool
	// fastLapStartTime     time.Time
	// fastLapNumber        int
	// recording            bool
	fastestLap *fastLapInfo
	// fastLapTime          time.Duration
	// prevDistanceToTarget float64
	// prevLocation         Messages.Location

	driverCurrentLaps map[int]*fastLapInfo

	compareDriverNum int

	startLine location

	sortedDriverNames []int
}

func CreateImproving() Panel {
	return &improving{}
}

func (i *improving) ProcessEventTime(data Messages.EventTime)                    {}
func (i *improving) ProcessEvent(data Messages.Event)                            {}
func (i *improving) ProcessWeather(data Messages.Weather)                        {}
func (i *improving) ProcessRadio(data Messages.Radio)                            {}
func (i *improving) ProcessTelemetry(data Messages.Telemetry)                    {}
func (i *improving) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (i *improving) Close()                                                      {}

func (i *improving) Type() Type { return QualifyingImproving }

func (i *improving) Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig) {
	i.dataSrc = dataSrc
	i.startLine = location{x: -386, y: 1170}
	i.driverCurrentLaps = make(map[int]*fastLapInfo)
}

func (i *improving) ProcessDrivers(data Messages.Drivers) {

	i.sortedDriverNames = []int{}

	for x := range data.Drivers {
		i.driverCurrentLaps[data.Drivers[x].Number] = &fastLapInfo{
			driverName:              data.Drivers[x].Name,
			driverNumber:            data.Drivers[x].Number,
			isRecording:             false,
			prevDistanceToStartLine: math.MaxFloat64,
		}

		i.sortedDriverNames = append(i.sortedDriverNames, data.Drivers[x].Number)

		if data.Drivers[x].ShortName == "SAI" {
			i.fastestDriverNum = data.Drivers[x].Number
			continue
		}

		if data.Drivers[x].ShortName == "HAM" {
			i.compareDriverNum = data.Drivers[x].Number
			continue
		}
	}

	i.sortedDriverNames = sort.IntSlice(i.sortedDriverNames)
}

func (i *improving) ProcessTiming(data Messages.Timing) {
	if len(i.driverCurrentLaps) == 0 {
		return
	}

	driverInfo := i.driverCurrentLaps[data.Number]

	driverInfo.location = data.Location
}

func (i *improving) ProcessLocation(data Messages.Location) {

	driverInfo := i.driverCurrentLaps[data.DriverNumber]

	// Only need to update if the driver is on a fast lap or outlap
	if driverInfo.location != Messages.OutLap && driverInfo.location != Messages.OnTrack {
		return
	}

	pos := location{x: data.X, y: data.Y}
	distToStart := distance(i.startLine, pos)

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

			// If getting closer to target
			if distToStart < driverInfo.prevDistanceToStartLine {
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
					i.fastestLap = driverInfo
				}

				// Reset driver tracking
				i.driverCurrentLaps[data.DriverNumber] = &fastLapInfo{
					markers:                 []timedLocation{},
					lapStartTime:            timeBetweenTwoPoints(i.startLine, driverInfo.prevLocation, data),
					prevDistanceToStartLine: math.MaxFloat64,
					prevLocation:            data,
					isRecording:             true,
					driverNumber:            driverInfo.driverNumber,
					driverName:              driverInfo.driverName,
				}

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
			currentMarker := driverInfo.markers[len(driverInfo.markers)-1]
			smallestDistance := math.MaxFloat64
			var smallestIndex int
			// Find the fastest lap point before to the most recent point
			for x := 0; x < len(i.fastestLap.markers); x++ {

				currentDistance := distance(i.fastestLap.markers[x].pos, currentMarker.pos)
				if currentDistance < smallestDistance {
					smallestDistance = currentDistance
					smallestIndex = x
				}
			}

			// TODO - do index range checks
			if smallestIndex == 0 || smallestIndex == len(i.fastestLap.markers)-1 {
				return
			}

			// Find the second closest point (is it before or after?e
			beforeDist := distance(i.fastestLap.markers[smallestIndex-1].pos, currentMarker.pos)
			afterDist := distance(i.fastestLap.markers[smallestIndex+1].pos, currentMarker.pos)

			// Current is between before and smallest
			var start, end timedLocation
			if beforeDist < afterDist {
				start = i.fastestLap.markers[smallestIndex-1]
				end = i.fastestLap.markers[smallestIndex]
			} else {
				start = i.fastestLap.markers[smallestIndex]
				end = i.fastestLap.markers[smallestIndex+1]
			}

			// Normalize the recent point back to the fastest point
			timeDiff := timeBetweenTwoPoints(pos,
				Messages.Location{X: start.pos.x, Y: start.pos.y, Timestamp: i.fastestLap.lapStartTime.Add(start.timestamp)},
				Messages.Location{X: end.pos.x, Y: end.pos.y, Timestamp: i.fastestLap.lapStartTime.Add(end.timestamp)})
			fastestElapsed := timeDiff.Sub(i.fastestLap.lapStartTime)

			// Update time diff
			elapsedLapTime := currentMarker.timestamp
			driverInfo.diffToFastest = elapsedLapTime - fastestElapsed
		}
	}
}

func (i *improving) Draw(width int, height int) []giu.Widget {
	if i.fastestLap == nil {
		return []giu.Widget{
			giu.Label("Waiting for first fast lap..."),
		}
	}

	results := []giu.Widget{}
	results = append(results, giu.Labelf("Fastest: %s %v", i.fastestLap.driverName, fmtDuration(i.fastestLap.estimatedLapTime)))

	for _, driverNum := range i.sortedDriverNames {
		driver := i.driverCurrentLaps[driverNum]
		if driver.location == Messages.OnTrack {
			results = append(results, giu.Labelf("%s - %v", driver.driverName, fmtDuration(driver.diffToFastest)))
		}
	}

	return results

	// return []giu.Widget{
	// 	giu.Labelf("Fast Driver On Quali Lap: %s", i.fastestLap.driverName),
	// 	giu.Labelf("Fast Lap Start Time: %v", i.fastestLap.lapStartTime),
	// 	giu.Labelf("Location Count: %d", len(i.fastestLap.markers)),
	// 	//		 giu.Labelf("Recording: %t", i.recording),
	// 	giu.Labelf("Fast Lap Time: %v", i.fastestLap.estimatedLapTime),
	// 	//		 giu.Labelf("Prev Distance To Target: %f", i.prevDistanceToTarget),
	// 	giu.Labelf("LeClerc Diff: %v", i.driverCurrentLaps[16].diffToFastest),
	// }
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
