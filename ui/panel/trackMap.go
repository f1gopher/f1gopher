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
	"image"
	"image/color"
	"sort"
	"sync"

	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/ungerik/go-cairo"
	"golang.org/x/image/colornames"
)

type trackMapInfo struct {
	color color.RGBA
	name  string
}

type trackMap struct {
	mapStore *trackMapStore

	driverData          map[int]trackMapInfo
	driverPositions     map[int]Messages.Location
	driverPositionsLock sync.Mutex

	trackTexture       *giu.Texture
	trackTextureWidth  float32
	trackTextureHeight float32
	mapGc              *cairo.Surface
	currentWidth       int
	currentHeight      int
}

const safetyCarDriverNum = 127

func CreateTrackMap() Panel {
	return &trackMap{
		mapStore:        CreateTrackMapStore(),
		driverPositions: map[int]Messages.Location{},
		driverData:      map[int]trackMapInfo{},
	}
}

func (t *trackMap) ProcessEventTime(data Messages.EventTime)                    {}
func (t *trackMap) ProcessEvent(data Messages.Event)                            {}
func (t *trackMap) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (t *trackMap) ProcessWeather(data Messages.Weather)                        {}
func (t *trackMap) ProcessRadio(data Messages.Radio)                            {}
func (t *trackMap) ProcessTelemetry(data Messages.Telemetry)                    {}
func (t *trackMap) Close()                                                      {}

func (t *trackMap) Type() Type { return TrackMap }

func (t *trackMap) Init(dataSrc f1gopherlib.F1GopherLib) {
	// Clear previous session data
	t.driverPositions = map[int]Messages.Location{}
	t.driverData = map[int]trackMapInfo{}
	t.mapGc = nil
	t.currentWidth = 0
	t.currentHeight = 0

	t.mapStore.SelectTrack(dataSrc.Track(), dataSrc.TrackYear())
}

func (t *trackMap) ProcessDrivers(data Messages.Drivers) {
	for x := range data.Drivers {
		t.driverData[data.Drivers[x].Number] = trackMapInfo{
			color: data.Drivers[x].Color,
			name:  data.Drivers[x].ShortName,
		}
	}
}

func (t *trackMap) ProcessLocation(data Messages.Location) {
	t.driverPositionsLock.Lock()
	t.driverPositions[data.DriverNumber] = data
	t.driverPositionsLock.Unlock()

	t.mapStore.ProcessLocation(data)
}

func (t *trackMap) ProcessTiming(data Messages.Timing) {
	t.mapStore.ProcessTiming(data)
}

func (t *trackMap) Draw(width int, height int) []giu.Widget {
	cars := []Messages.Location{}
	t.driverPositionsLock.Lock()
	for _, x := range t.driverPositions {
		cars = append(cars, x)
	}
	t.driverPositionsLock.Unlock()

	t.redraw(width, height, cars)

	if t.trackTexture != nil {
		return []giu.Widget{
			giu.Image(t.trackTexture).Size(t.trackTextureWidth, t.trackTextureHeight),
		}
	}

	return []giu.Widget{
		giu.Custom(func() {
			canvas := giu.GetCanvas()
			pos := giu.GetCursorScreenPos()

			textWidth, _ := giu.CalcTextSize("Building Map...")
			offset := int(textWidth / 2)
			canvas.AddText(pos.Add(image.Pt((width/2)-offset, height/2)), colornames.Yellow, "Building Map...")
		}),
	}
}

func (t *trackMap) redraw(width int, height int, cars []Messages.Location) {
	// Widget has a margin the image needs to fit in
	displayWidth := width - 16
	displayHeight := height - 16
	available, scaling, xOffset, yOffset := t.mapStore.MapAvailable(displayWidth, displayHeight)

	if available {
		if t.mapGc == nil || displayWidth != t.currentWidth || displayHeight != t.currentHeight {
			t.mapGc = cairo.NewSurface(cairo.FORMAT_ARGB32, displayWidth, displayHeight)
			t.mapGc.SelectFontFace("sans-serif", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
			t.mapGc.SetFontSize(10.0)
			t.currentWidth = width
			t.currentHeight = height
			t.trackTextureWidth = float32(displayWidth)
			t.trackTextureHeight = float32(displayHeight)
		}

		// Overwrite the previous data with a clean track outline
		t.mapGc.SetData(t.mapStore.gc.GetData())

		// Sort into a consistent order for drawing so we don't get flickr when drivers are overlapping
		sort.Slice(cars, func(i, j int) bool {
			return cars[i].DriverNumber < cars[j].DriverNumber
		})

		for _, car := range cars {
			x := car.X
			y := car.Y

			// Invert x
			x = float64(displayWidth) - x

			x = x / scaling
			y = y / scaling

			x += float64(xOffset)
			y += float64(yOffset)

			driverInfo, exists := t.driverData[car.DriverNumber]
			driverColor := colornames.White
			driverName := "UNK"
			if exists {
				driverColor = driverInfo.color
				driverName = driverInfo.name
			} else if car.DriverNumber == safetyCarDriverNum {
				// We don't have driver data for the safety car but once it goes on track we get position info for it
				driverName = "SC"
			}

			// Draw marker
			t.mapGc.SetSourceRGBA(float64(driverColor.R)/255.0, float64(driverColor.G)/255.0, float64(driverColor.B)/255.0, 1.0)
			t.mapGc.Rectangle(x-5, y-5, 10, 10)
			t.mapGc.Fill()
			t.mapGc.Stroke()

			// Draw driver short name
			t.mapGc.MoveTo(x+float64(15), y+2.5)
			t.mapGc.ShowText(driverName)
			t.mapGc.Stroke()
		}

		giu.EnqueueNewTextureFromRgba(t.mapGc.GetImage(), func(texture *giu.Texture) {
			t.trackTexture = texture
		})
	}
}
