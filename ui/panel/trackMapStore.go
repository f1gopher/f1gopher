package panel

import (
	"fmt"
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib/Messages"
	"golang.org/x/image/colornames"
	"image"
	"math"
	"os"
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
}

type trackMapStore struct {
	tracks map[string]*trackInfo

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
}

func CreateTrackMapStore() *trackMapStore {
	store := &trackMapStore{
		tracks:       make(map[string]*trackInfo),
		currentTrack: nil,
		trackReady:   false,
		pitlaneReady: false,
	}

	// Load known tracks
	for x := range trackMapData {
		store.tracks[trackMapData[x].name] = &trackMapData[x]
	}

	return store
}

func (t *trackMapStore) SelectTrack(name string, year int) {
	track, exists := t.tracks[name]
	if exists {
		t.currentTrack = track
		t.trackReady = true
		t.pitlaneReady = true
		t.targetDriver = 0
		return
	}

	t.currentTrack = &trackInfo{
		name:        name,
		yearCreated: year,
		outline:     make([]image.Point, 0),
		pitlane:     make([]image.Point, 0),
	}
	t.trackReady = false
	t.pitlaneReady = false

	t.locations = make([]Messages.Location, 0)
	t.trackStart = time.Time{}
	t.trackEnd = time.Time{}
	t.pitlaneStart = time.Time{}
	t.pitlaneEnd = time.Time{}
	t.prevLocation = Messages.NoLocation
	t.targetDriver = 0
}

func (t *trackMapStore) MapAvailable(width int, height int) (available bool, scaling float64, xOffset int, yOffset int) {
	if t.trackReady {
		xRange := float64(t.currentTrack.maxX - t.currentTrack.minX)
		yRange := float64(t.currentTrack.maxY - t.currentTrack.minY)

		// TODO - use actual panel size
		// TODO - handle panel resizing

		// Add 0.5 to round up
		if xRange > yRange {
			t.currentTrack.scaling = xRange / float64(width)
		} else {
			t.currentTrack.scaling = yRange / float64(height)
		}

		// TODO - this doesn't always seem right. For Abu Dhabi X it wrong but Y is right
		t.currentTrack.xOffset = width - int((math.Abs(float64(t.currentTrack.minX))/xRange)*float64(width))
		t.currentTrack.yOffset = int((math.Abs(float64(t.currentTrack.minY)) / yRange) * float64(height))

		return t.trackReady, t.currentTrack.scaling, t.currentTrack.xOffset, t.currentTrack.yOffset
	}

	return false, 0.0, 0, 0
}

func (t *trackMapStore) DrawMap(canvas *giu.Canvas, pos image.Point, width int) {
	if !t.trackReady {
		panic("No map available to draw")
	}

	if t.pitlaneReady {
		canvas.PathClear()
		for loc := range t.currentTrack.pitlane {
			xPoint := float64(t.currentTrack.pitlane[loc].X)
			xPoint = float64(width) - xPoint

			x := (xPoint / t.currentTrack.scaling) + float64(t.currentTrack.xOffset+pos.X)
			y := (float64(t.currentTrack.pitlane[loc].Y) / t.currentTrack.scaling) + float64(t.currentTrack.yOffset+pos.Y)
			canvas.PathLineTo(image.Pt(int(x+0.5), int(y+0.5)))
		}
		canvas.PathStroke(colornames.White, false, 1)
	}

	canvas.PathClear()
	for loc := range t.currentTrack.outline {
		xPoint := float64(t.currentTrack.outline[loc].X)
		xPoint = float64(width) - xPoint

		x := (xPoint / t.currentTrack.scaling) + float64(t.currentTrack.xOffset+pos.X)
		y := (float64(t.currentTrack.outline[loc].Y) / t.currentTrack.scaling) + float64(t.currentTrack.yOffset+pos.Y)
		canvas.PathLineTo(image.Pt(int(x+0.5), int(y+0.5)))
	}
	canvas.PathStroke(colornames.Yellow, true, 1)
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

	if t.targetDriver == 0 {
		t.targetDriver = data.Number
	}

	if data.Number == t.targetDriver {

		if t.trackStart.IsZero() && data.Location == Messages.OnTrack {
			t.trackStart = data.Timestamp
			t.recordingLap = data.Lap
			//return
		}

		// If dive into pits when recording track then abort record track
		if !t.trackStart.IsZero() && t.trackEnd.IsZero() && data.Location == Messages.Pitlane {
			t.trackStart = time.Time{}
			t.recordingLap = -1
		}

		if !t.trackStart.IsZero() && t.trackEnd.IsZero() && data.Lap == t.recordingLap+1 {
			t.trackEnd = data.Timestamp
			t.recordingLap = -1
			//return
		}

		if t.pitlaneStart.IsZero() && data.Location == Messages.Pitlane &&
			(t.prevLocation == Messages.OnTrack || t.prevLocation == Messages.OutLap) {

			t.pitlaneStart = data.Timestamp
			//return
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

			for _, location := range t.locations {
				if location.Timestamp.Before(t.trackStart) {
					continue
				}

				t.currentTrack.outline = append(t.currentTrack.outline, image.Pt(int(location.X), int(location.Y)))

				if location.Timestamp.After(t.trackEnd) {
					break
				}
			}

			t.currentTrack.minX = math.MaxInt
			t.currentTrack.maxX = math.MinInt
			t.currentTrack.minY = math.MaxInt
			t.currentTrack.maxY = math.MinInt

			// TODO - smooth points

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

			// Store track for later use
			t.tracks[t.currentTrack.name] = t.currentTrack
		}

		if !t.pitlaneReady &&
			!t.pitlaneStart.IsZero() &&
			!t.pitlaneEnd.IsZero() {

			t.pitlaneReady = true

			// Count back
			actualPitStart := t.pitlaneStart.Add(-7 * time.Second)

			for _, location := range t.locations {
				if location.Timestamp.Before(actualPitStart) {
					continue
				}

				t.currentTrack.pitlane = append(t.currentTrack.pitlane, image.Pt(int(location.X), int(location.Y)))

				if location.Timestamp.After(t.pitlaneEnd) {
					break
				}
			}

			// Store track for later use
			t.tracks[t.currentTrack.name] = t.currentTrack
		}
	}
}

func (t *trackMapStore) writeToFile(file string) {
	f, _ := os.Create(file)
	defer f.Close()

	f.WriteString(`package panel

import (
	"image"
)

var trackMapData = []trackInfo{
`)

	for x := range t.tracks {
		f.WriteString("\t{\n")
		f.WriteString(fmt.Sprintf("\t\tname: \"%s\",\n", t.tracks[x].name))
		f.WriteString(fmt.Sprintf("\t\tyearCreated: \"%d\",\n", t.tracks[x].yearCreated))
		f.WriteString("\t\toutline: []image.Point{\n")
		for _, p := range t.tracks[x].outline {
			f.WriteString(fmt.Sprintf("\t\t\timage.Pt(%d, %d),\n", p.X, p.Y))
		}
		f.WriteString("\t\t},\n")
		f.WriteString("\t\tpitlane: []image.Point{\n")
		for _, p := range t.tracks[x].pitlane {
			f.WriteString(fmt.Sprintf("\t\t\timage.Pt(%d, %d),\n", p.X, p.Y))
		}
		f.WriteString("\t\t},\n")
		f.WriteString(fmt.Sprintf("\t\tminX: %d,\n", t.tracks[x].minX))
		f.WriteString(fmt.Sprintf("\t\tmaxX: %d,\n", t.tracks[x].maxX))
		f.WriteString(fmt.Sprintf("\t\tminY: %d,\n", t.tracks[x].minY))
		f.WriteString(fmt.Sprintf("\t\tmaxY: %d,\n", t.tracks[x].maxY))
		f.WriteString("\t},\n")
	}

	f.WriteString("}")
}
