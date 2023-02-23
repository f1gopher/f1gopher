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
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/ungerik/go-cairo"
	"golang.org/x/image/colornames"
	"image"
	"image/color"
	"math"
	"os"
	"sort"
	"time"
)

type trackInfo struct {
	name        string
	yearCreated int
	outline     []image.Point
	pitlane     []image.Point
	scaling     float64
	xOffset     int
	yOffset     int
	minX        int
	maxX        int
	minY        int
	maxY        int
	rotation    float64
}

type trackMapStore struct {
	tracks map[string][]*trackInfo

	currentTrack *trackInfo

	trackReady   bool
	pitlaneReady bool
	recordingLap int
	targetDriver int

	locations    []Messages.Location
	trackStart   time.Time
	trackEnd     time.Time
	pitlaneStart time.Time
	pitlaneEnd   time.Time
	prevLocation Messages.CarLocation

	backgroundColor color.RGBA
	currentWidth    int
	currentHeight   int
	gc              *cairo.Surface
}

func CreateTrackMapStore() *trackMapStore {
	store := &trackMapStore{
		tracks:        make(map[string][]*trackInfo),
		currentTrack:  nil,
		trackReady:    false,
		pitlaneReady:  false,
		currentWidth:  0,
		currentHeight: 0,
		// Transparent background by default
		backgroundColor: color.RGBA{R: 0, G: 0, B: 0, A: 0},
	}

	// Load known tracks
	for x := range trackMapData {
		store.storeMap(&trackMapData[x])
	}

	return store
}

func (t *trackMapStore) SelectTrack(name string, year int) {
	trackVersions, exists := t.tracks[name]
	if exists {
		for x := range trackVersions {
			if year == trackVersions[x].yearCreated {
				t.currentTrack = trackVersions[x]
				t.trackReady = true
				t.pitlaneReady = true
				t.targetDriver = 0
				t.currentWidth = 0
				t.currentHeight = 0
				if t.gc != nil {
					t.gc.Destroy()
					t.gc = nil
				}
				return
			}
		}

		// No track for the year we need
	}

	t.currentTrack = &trackInfo{
		name:        name,
		yearCreated: year,
		outline:     make([]image.Point, 0),
		pitlane:     make([]image.Point, 0),
		rotation:    0,
	}
	t.trackReady = false
	t.pitlaneReady = false
	t.currentWidth = 0
	t.currentHeight = 0
	if t.gc != nil {
		t.gc.Destroy()
		t.gc = nil
	}

	t.locations = make([]Messages.Location, 0)
	t.trackStart = time.Time{}
	t.trackEnd = time.Time{}
	t.pitlaneStart = time.Time{}
	t.pitlaneEnd = time.Time{}
	t.prevLocation = Messages.NoLocation
	t.targetDriver = 0
}

func (t *trackMapStore) MapAvailable(width int, height int) (available bool, scaling float64, xOffset int, yOffset int, rotation float64) {
	if t.trackReady {
		// If the width and height haven't changed since the last time we drew the track outline
		// then don't redraw just return
		if t.gc != nil && width == t.currentWidth && height == t.currentHeight {
			return t.trackReady, t.currentTrack.scaling, t.currentTrack.xOffset, t.currentTrack.yOffset, t.currentTrack.rotation
		}

		t.currentWidth = width
		t.currentHeight = height

		xRange := float64(t.currentTrack.maxX - t.currentTrack.minX)
		yRange := float64(t.currentTrack.maxY - t.currentTrack.minY)

		const border = 50

		// Pick the best scaling option to fill the display
		var a, b float64
		// If we rotate width and height are swapped
		if t.currentTrack.name != "Bahrain International Circuit" {
			a = xRange / float64(width-border)
			b = yRange / float64(height-border)

			if a > b {
				t.currentTrack.scaling = a
			} else {
				t.currentTrack.scaling = b
			}

			// X
			mapCentreOffsetX := int(float64(t.currentTrack.maxX) / t.currentTrack.scaling)
			mapMarginX := int((float64(width) - (xRange / t.currentTrack.scaling)) / 2)
			t.currentTrack.xOffset = mapCentreOffsetX + mapMarginX

			// Y
			mapCentreOffsetY := int(float64(t.currentTrack.maxY) / t.currentTrack.scaling)
			mapMarginY := int((float64(height) - (yRange / t.currentTrack.scaling)) / 2)
			t.currentTrack.yOffset = height - (mapCentreOffsetY + mapMarginY)
		} else {
			a = yRange / float64(width-border)
			b = xRange / float64(height-border)

			if a > b {
				t.currentTrack.scaling = a
			} else {
				t.currentTrack.scaling = b
			}

			// X
			mapCentreOffsetX := int(float64(t.currentTrack.maxY) / t.currentTrack.scaling)
			mapMarginX := int((float64(width) - (yRange / t.currentTrack.scaling)) / 2)
			t.currentTrack.xOffset = mapCentreOffsetX + mapMarginX

			// Y
			mapCentreOffsetY := int(float64(t.currentTrack.maxX) / t.currentTrack.scaling)
			mapMarginY := int((float64(height) - (xRange / t.currentTrack.scaling)) / 2)
			t.currentTrack.yOffset = (height - (mapCentreOffsetY + mapMarginY)) * 2
		}

		if t.gc != nil {
			t.gc.Destroy()
		}
		t.gc = cairo.NewSurface(cairo.FORMAT_ARGB32, width, height)
		t.gc.SetSourceRGBA(
			float64(t.backgroundColor.R)/255.0,
			float64(t.backgroundColor.G)/255.0,
			float64(t.backgroundColor.B)/255.0,
			float64(t.backgroundColor.A)/255.0)
		t.gc.Rectangle(0, 0, float64(t.gc.GetWidth()), float64(t.gc.GetHeight()))
		t.gc.Fill()

		t.gc.SetSourceRGBA(
			1,
			0,
			0,
			1)

		// Rotate around the centre of the track
		t.gc.Translate(float64(width)/2, float64(height)/2)
		t.gc.Rotate(t.currentTrack.rotation)

		// Center marker
		//t.gc.NewPath()
		//t.gc.MoveTo(-100, 0)
		//t.gc.LineTo(100, 0)
		//t.gc.MoveTo(0, -100)
		//t.gc.LineTo(0, 100)
		//t.gc.Stroke()

		t.gc.Translate(-float64(width)/2, -float64(height)/2)

		// Pitlane
		color := colornames.Yellow
		t.gc.SetSourceRGBA(float64(color.R)/255.0, float64(color.G)/255.0, float64(color.B)/255.0, 1.0)
		t.gc.NewPath()
		first := true

		for loc := range t.currentTrack.pitlane {
			xPoint := float64(t.currentTrack.pitlane[loc].X)
			xPoint = float64(width) - xPoint

			x := (xPoint / t.currentTrack.scaling) + float64(t.currentTrack.xOffset)
			y := (float64(t.currentTrack.pitlane[loc].Y) / t.currentTrack.scaling) + float64(t.currentTrack.yOffset)

			if first {
				t.gc.MoveTo(x, y)
				first = false
				continue
			}

			t.gc.LineTo(x, y)
		}
		t.gc.Stroke()

		// Track
		color = colornames.White
		t.gc.SetSourceRGBA(float64(color.R)/255.0, float64(color.G)/255.0, float64(color.B)/255.0, 1.0)
		t.gc.NewPath()
		first = true

		for loc := range t.currentTrack.outline {
			xPoint := float64(t.currentTrack.outline[loc].X)
			xPoint = float64(width) - xPoint

			x := (xPoint / t.currentTrack.scaling) + float64(t.currentTrack.xOffset)
			y := (float64(t.currentTrack.outline[loc].Y) / t.currentTrack.scaling) + float64(t.currentTrack.yOffset)

			if first {
				t.gc.MoveTo(x, y)
				first = false
				continue
			}

			t.gc.LineTo(x, y)
		}
		t.gc.ClosePath()
		t.gc.Stroke()

		t.gc.Flush()

		return t.trackReady, t.currentTrack.scaling, t.currentTrack.xOffset, t.currentTrack.yOffset, t.currentTrack.rotation
	}

	return false, 0.0, 0, 0, 0
}

func (t *trackMapStore) ProcessLocation(data Messages.Location) {
	if t.trackReady && t.pitlaneReady {
		return
	}

	if data.DriverNumber == t.targetDriver {

		if len(t.locations) == 0 {
			t.locations = append(t.locations, data)
		} else {
			last := t.locations[len(t.locations)-1]

			if !(math.Abs(last.X-data.X) < 0.00001 && math.Abs(last.Y-data.Y) < 0.00001) {
				t.locations = append(t.locations, data)
			}
		}
	}
}

func (t *trackMapStore) ProcessTiming(data Messages.Timing) {
	if t.trackReady && t.pitlaneReady {
		return
	}

	if t.targetDriver == 0 && data.Number != 0 && data.Location == Messages.OutLap {
		t.targetDriver = data.Number
	}

	if data.Number == t.targetDriver {

		if t.trackStart.IsZero() && data.Location == Messages.OnTrack && data.Lap != 1 &&
			data.Sector1 != 0 && data.Sector2 != 0 && data.Sector3 != 0 {

			t.trackStart = data.Timestamp
			t.recordingLap = data.Lap
		}

		// If dive into pits when recording track then abort record track
		if !t.trackStart.IsZero() && t.trackEnd.IsZero() && data.Location == Messages.Pitlane {
			t.trackStart = time.Time{}
			t.recordingLap = -1
		}

		// If the car stops reset and use another driver
		if data.Location == Messages.OutOfRace || data.Location == Messages.Stopped {
			t.targetDriver = 0
			t.trackStart = time.Time{}
			t.recordingLap = -1
		}

		if !t.trackStart.IsZero() && t.trackEnd.IsZero() && data.Lap == t.recordingLap+1 &&
			data.Sector1 != 0 && data.Sector2 != 0 && data.Sector3 != 0 {
			t.trackEnd = data.Timestamp
			t.recordingLap = -1
		}

		if t.pitlaneStart.IsZero() && data.Location == Messages.Pitlane &&
			(t.prevLocation == Messages.OnTrack || t.prevLocation == Messages.OutLap) {

			t.pitlaneStart = data.Timestamp
		}

		if !t.pitlaneStart.IsZero() && t.pitlaneEnd.IsZero() && data.Location == Messages.OutLap {
			t.pitlaneEnd = data.Timestamp
		}

		if t.prevLocation != data.Location {
			t.prevLocation = data.Location
		}

		if !t.trackReady &&
			!t.trackStart.IsZero() &&
			!t.trackEnd.IsZero() {

			t.trackReady = true

			for x, location := range t.locations {
				if location.Timestamp.Before(t.trackStart) {
					continue
				}

				// Always include the first and last and then every third location
				if x%3 == 0 || x == 0 || x == len(t.locations)-1 {
					t.currentTrack.outline = append(t.currentTrack.outline, image.Pt(int(location.X), int(location.Y)))
				}

				if location.Timestamp.After(t.trackEnd) {
					break
				}
			}

			t.currentTrack.minX = math.MaxInt
			t.currentTrack.maxX = math.MinInt
			t.currentTrack.minY = math.MaxInt
			t.currentTrack.maxY = math.MinInt

			for x := range t.currentTrack.outline {
				if t.currentTrack.outline[x].X < t.currentTrack.minX {
					t.currentTrack.minX = t.currentTrack.outline[x].X
				}
				if t.currentTrack.outline[x].X > t.currentTrack.maxX {
					t.currentTrack.maxX = t.currentTrack.outline[x].X
				}

				if t.currentTrack.outline[x].Y < t.currentTrack.minY {
					t.currentTrack.minY = t.currentTrack.outline[x].Y
				}
				if t.currentTrack.outline[x].Y > t.currentTrack.maxY {
					t.currentTrack.maxY = t.currentTrack.outline[x].Y
				}
			}

			if len(t.currentTrack.outline) == 0 {
				t.trackReady = false
				t.targetDriver = 0
				t.trackStart = time.Time{}
				t.trackEnd = time.Time{}
				t.recordingLap = -1
			}
		}

		if !t.pitlaneReady &&
			!t.pitlaneStart.IsZero() &&
			!t.pitlaneEnd.IsZero() {

			t.pitlaneReady = true

			// Count back
			actualPitStart := t.pitlaneStart.Add(-7 * time.Second)

			for x, location := range t.locations {
				if location.Timestamp.Before(actualPitStart) {
					continue
				}

				// Always include the first and last and then every third location
				if x%3 == 0 || x == 0 || x == len(t.locations)-1 {
					t.currentTrack.pitlane = append(t.currentTrack.pitlane, image.Pt(int(location.X), int(location.Y)))
				}

				if location.Timestamp.After(t.pitlaneEnd) {
					break
				}
			}
		}

		if t.trackReady && t.pitlaneReady {
			// Store track for later use
			t.storeMap(t.currentTrack)
		}
	}
}

func (t *trackMapStore) storeMap(trackMap *trackInfo) {
	_, exists := t.tracks[trackMap.name]
	if exists {
		t.tracks[trackMap.name] = append(t.tracks[trackMap.name], trackMap)
	} else {
		t.tracks[trackMap.name] = []*trackInfo{trackMap}
	}
}

func (t *trackMapStore) writeToFile(file string) {
	f, _ := os.Create(file)
	defer f.Close()

	f.WriteString(`// F1Gopher - Copyright (C) 2023 f1gopher
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
	"image"
)

var trackMapData = []trackInfo{
`)

	// Sort tracks into alphabetical order so they are written to disk in a consistent way
	var trackNames []string
	for name := range t.tracks {
		trackNames = append(trackNames, name)
	}
	sort.Strings(trackNames)

	for _, x := range trackNames {
		for _, track := range t.tracks[x] {
			f.WriteString("\t{\n")
			f.WriteString(fmt.Sprintf("\t\tname: \"%s\",\n", track.name))
			f.WriteString(fmt.Sprintf("\t\tyearCreated: %d,\n", track.yearCreated))
			f.WriteString("\t\toutline: []image.Point{\n")
			for _, p := range track.outline {
				f.WriteString(fmt.Sprintf("\t\t\timage.Pt(%d, %d),\n", p.X, p.Y))
			}
			f.WriteString("\t\t},\n")
			f.WriteString("\t\tpitlane: []image.Point{\n")
			for _, p := range track.pitlane {
				f.WriteString(fmt.Sprintf("\t\t\timage.Pt(%d, %d),\n", p.X, p.Y))
			}
			f.WriteString("\t\t},\n")
			f.WriteString(fmt.Sprintf("\t\tminX: %d,\n", track.minX))
			f.WriteString(fmt.Sprintf("\t\tmaxX: %d,\n", track.maxX))
			f.WriteString(fmt.Sprintf("\t\tminY: %d,\n", track.minY))
			f.WriteString(fmt.Sprintf("\t\tmaxY: %d,\n", track.maxY))
			f.WriteString(fmt.Sprintf("\t\trotation: %f,\n", track.rotation))
			f.WriteString("\t},\n")
		}
	}

	f.WriteString("}")
}
