package panel

import (
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/ungerik/go-cairo"
	"golang.org/x/image/colornames"
	"image"
	"image/color"
	"sync"
)

type trackMap struct {
	mapStore *trackMapStore

	driverColors        map[int]color.RGBA
	driverNames         map[int]string
	driverPositions     map[int]Messages.Location
	driverPositionsLock sync.Mutex

	trackTexture  *giu.Texture
	mapGc         *cairo.Surface
	currentWidth  int
	currentHeight int

	widget *giu.ImageWithRgbaWidget
}

func CreateTrackMap() Panel {
	return &trackMap{
		mapStore:        CreateTrackMapStore(),
		driverColors:    map[int]color.RGBA{},
		driverPositions: map[int]Messages.Location{},
	}
}

func (t *trackMap) Close()                                                      {}
func (t *trackMap) ProcessEventTime(data Messages.EventTime)                    {}
func (t *trackMap) ProcessEvent(data Messages.Event)                            {}
func (t *trackMap) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (t *trackMap) ProcessWeather(data Messages.Weather)                        {}
func (t *trackMap) ProcessRadio(data Messages.Radio)                            {}

func (t *trackMap) Init(dataSrc f1gopherlib.F1GopherLib) {
	// Clear previous session data
	t.driverPositions = map[int]Messages.Location{}
	t.driverColors = map[int]color.RGBA{}
	t.driverNames = map[int]string{}
	t.mapGc = nil
	t.currentWidth = 0
	t.currentWidth = 0

	t.mapStore.SelectTrack(dataSrc.Track(), dataSrc.TrackYear())
}

func (t *trackMap) ProcessLocation(data Messages.Location) {
	t.driverPositionsLock.Lock()
	t.driverPositions[data.DriverNumber] = data
	t.driverPositionsLock.Unlock()

	t.mapStore.ProcessLocation(data)
}

func (t *trackMap) ProcessTiming(data Messages.Timing) {

	// TODO - do this properly
	if len(t.driverColors) < 20 {
		t.driverColors[data.Number] = data.Color
		t.driverNames[data.Number] = data.ShortName
	}

	t.mapStore.ProcessTiming(data)
}

func (t *trackMap) Draw() (title string, widgets []giu.Widget) {
	width := 500
	height := 500

	cars := []Messages.Location{}
	t.driverPositionsLock.Lock()
	for _, x := range t.driverPositions {
		cars = append(cars, x)
	}
	t.driverPositionsLock.Unlock()

	displayWidth := width - 15
	displayHeight := height - 15
	available, scaling, xOffset, yOffset := t.mapStore.MapAvailable(displayWidth, displayHeight)

	if available {
		if displayWidth != t.currentWidth && displayHeight != t.currentHeight {
			t.mapGc = cairo.NewSurface(cairo.FORMAT_ARGB32, displayWidth, displayHeight)
			t.mapGc.SelectFontFace("sans-serif", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
			t.mapGc.SetFontSize(10.0)
			t.currentWidth = width
			t.currentHeight = height
		}

		// Overwrite the previous data with a clean track outline
		t.mapGc.SetData(t.mapStore.gc.GetData())

		for _, car := range cars {
			x := car.X
			y := car.Y

			// Invert x
			x = float64(displayWidth) - x

			x = x / scaling
			y = y / scaling

			x += float64(xOffset)
			y += float64(yOffset)

			driverColor, exists := t.driverColors[car.DriverNumber]
			if !exists {
				driverColor = colornames.White
			}

			// Draw marker
			t.mapGc.SetSourceRGBA(float64(driverColor.R)/255.0, float64(driverColor.G)/255.0, float64(driverColor.B)/255.0, 1.0)
			t.mapGc.Rectangle(x-5, y-5, 10, 10)
			t.mapGc.Fill()
			t.mapGc.Stroke()

			// Draw driver short name
			t.mapGc.MoveTo(x+float64(15), y+2.5)
			t.mapGc.ShowText(t.driverNames[car.DriverNumber])
			t.mapGc.Stroke()
		}

		t.mapGc.Finish()

		giu.EnqueueNewTextureFromRgba(t.mapGc.GetImage(), func(texture *giu.Texture) {
			t.trackTexture = texture
		})

		return "Track Map", []giu.Widget{
			giu.Image(t.trackTexture).Size(float32(displayWidth), float32(displayHeight)),
		}
	}

	return "Track Map", []giu.Widget{
		giu.Custom(func() {
			canvas := giu.GetCanvas()
			pos := giu.GetCursorScreenPos()

			textWidth, _ := giu.CalcTextSize("Building Map...")
			offset := int(textWidth / 2)
			canvas.AddText(pos.Add(image.Pt((width/2)-offset, height/2)), colornames.Yellow, "Building Map...")
		}),
	}
}
