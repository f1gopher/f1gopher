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
	"image/draw"
	"math"
	"sort"
	"sync"

	"github.com/AllenDang/imgui-go"

	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/ungerik/go-cairo"
	"golang.org/x/image/colornames"
)

type trackMapInfo struct {
	color     color.RGBA
	name      string
	isStopped bool
}

type trackMap struct {
	mapStore *trackMapStore

	driverData          map[int]trackMapInfo
	driverPositions     map[int]Messages.Location
	driverPositionsLock sync.Mutex
	event               Messages.Event
	eventLock           sync.Mutex
	showStoppedCars     bool

	trackTexture       imgui.TextureID
	trackTextureWidth  float32
	trackTextureHeight float32
	mapGc              *cairo.Surface
	currentWidth       int
	currentHeight      int
}

const safetyCarDriverNum = 127

func CreateTrackMap(trackMaps *trackMapStore) Panel {
	return &trackMap{
		mapStore:        trackMaps,
		driverPositions: map[int]Messages.Location{},
		driverData:      map[int]trackMapInfo{},
		showStoppedCars: true,
	}
}

func (t *trackMap) ProcessEventTime(data Messages.EventTime)                    {}
func (t *trackMap) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (t *trackMap) ProcessWeather(data Messages.Weather)                        {}
func (t *trackMap) ProcessRadio(data Messages.Radio)                            {}
func (t *trackMap) ProcessTelemetry(data Messages.Telemetry)                    {}
func (t *trackMap) Close()                                                      {}

func (t *trackMap) Type() Type { return TrackMap }

func (t *trackMap) Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig) {
	// Clear previous session data
	t.driverPositions = map[int]Messages.Location{}
	t.driverData = map[int]trackMapInfo{}
	if t.mapGc != nil {
		t.mapGc.Destroy()
		t.mapGc = nil
	}
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

	// Update the driver stopped state if it has changce
	driverData := t.driverData[data.Number]
	if data.Location == Messages.Stopped != driverData.isStopped {
		driverData.isStopped = data.Location == Messages.Stopped
		t.driverData[data.Number] = driverData
	}
}

func (t *trackMap) ProcessEvent(data Messages.Event) {
	t.mapStore.ProcessEvent(data)

	t.eventLock.Lock()
	t.event = data
	t.eventLock.Unlock()
}

func (t *trackMap) Draw(width int, height int) []giu.Widget {
	cars := []Messages.Location{}
	t.driverPositionsLock.Lock()
	for _, x := range t.driverPositions {
		cars = append(cars, x)
	}
	t.driverPositionsLock.Unlock()

	// Allow room for the checkbox
	height = height - 20

	t.redraw(width, height, cars)

	if t.trackTexture != 0 {
		return []giu.Widget{
			giu.Image(giu.ToTexture(t.trackTexture)).Size(t.trackTextureWidth, t.trackTextureHeight),
			giu.Row(
				giu.Checkbox("Show Stopped Cars", &t.showStoppedCars),
			),
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
		giu.Row(
			giu.Checkbox("Show Stopped Cars", &t.showStoppedCars),
		),
	}
}

func (t *trackMap) redraw(width int, height int, cars []Messages.Location) {
	// Widget has a margin the image needs to fit in
	displayWidth := width - 16
	displayHeight := height - 16
	available, scaling, xOffset, yOffset, rotation := t.mapStore.MapAvailable(displayWidth, displayHeight)

	if available {
		if t.mapGc == nil || displayWidth != t.currentWidth || displayHeight != t.currentHeight {
			if t.mapGc != nil {
				t.mapGc.Destroy()
			}
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

		// 0, 0 is the center of the display
		t.mapGc.Translate(float64(displayWidth)/2, float64(displayHeight)/2)

		// Center marker
		//t.mapGc.SetSourceRGBA(0, 1, 0, 1)
		//t.mapGc.NewPath()
		//t.mapGc.MoveTo(-50, 0)
		//t.mapGc.LineTo(50, 0)
		//t.mapGc.MoveTo(0, -50)
		//t.mapGc.LineTo(0, 50)
		//t.mapGc.Stroke()

		s := math.Sin(rotation)
		c := math.Cos(rotation)

		for _, car := range cars {
			x := car.X
			y := car.Y

			xVal := -(x - float64(xOffset))
			yVal := y - float64(yOffset)

			x = xVal*c - yVal*s
			y = xVal*s + yVal*c

			x = x / scaling
			y = y / scaling

			driverInfo, exists := t.driverData[car.DriverNumber]

			// Skip stopped cars if turned off
			if driverInfo.isStopped && !t.showStoppedCars {
				continue
			}

			driverColor := colornames.White
			driverName := "UNK"
			if exists {
				driverColor = driverInfo.color
				driverName = driverInfo.name
			} else if car.DriverNumber == safetyCarDriverNum {
				// If we are not under safety car conditions then don't display the safety car on the map
				t.eventLock.Lock()
				if t.event.SafetyCar != Messages.SafetyCar && t.event.SafetyCar != Messages.SafetyCarEnding {
					t.eventLock.Unlock()
					continue
				}
				t.eventLock.Unlock()

				// We don't have driver data for the safety car but once it goes on track we get position info for it
				driverName = "SC"
			}

			// Draw marker
			t.mapGc.SetSourceRGBA(float64(driverColor.R)/255.0, float64(driverColor.G)/255.0, float64(driverColor.B)/255.0, 1.0)
			t.mapGc.Rectangle(x-5, y-5, 10, 10)
			t.mapGc.Fill()
			t.mapGc.Stroke()

			// Draw driver short name
			t.mapGc.MoveTo(x+float64(10), y-5)
			t.mapGc.ShowText(driverName)
			t.mapGc.Stroke()
		}

		//// Finish/timing end line
		//s = math.Sin(t.mapStore.currentTrack.finishLineRotation)
		//c = math.Cos(t.mapStore.currentTrack.finishLineRotation)
		//startX := -(t.mapStore.currentTrack.finishLine.x - float64(xOffset))
		//startY := t.mapStore.currentTrack.finishLine.y - float64(yOffset)
		//startX = startX*c - startY*s
		//startY = startX*s + startY*c
		//startX = startX / scaling
		//startY = startY / scaling
		//t.mapGc.SetSourceRGBA(1, 0, 0, 1)
		//// t.mapGc.Rectangle(startX-10, startY-10, 20, 20)
		//// t.mapGc.Fill()
		//t.mapGc.MoveTo(startX, startY-10)
		//t.mapGc.LineTo(startX, startY+10)
		//t.mapGc.Stroke()

		// Convert image to texture and release any previous texture
		trueImg := t.mapGc.GetImage()
		rgba := image.NewRGBA(trueImg.Bounds())
		draw.Draw(rgba, trueImg.Bounds(), trueImg, image.Pt(0, 0), draw.Src)
		giu.Context.GetRenderer().ReleaseImage(t.trackTexture)
		t.trackTexture, _ = giu.Context.GetRenderer().LoadImage(rgba)
	}
}
