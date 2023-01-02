package panel

import (
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"golang.org/x/image/colornames"
	"image"
	"image/color"
	"sync"
)

type trackMap struct {
	mapStore *trackMapStore

	driverColors        map[int]color.Color
	driverNames         map[int]string
	driverPositions     map[int]Messages.Location
	driverPositionsLock sync.Mutex

	trackTexture *giu.Texture

	widget *giu.ImageWithRgbaWidget
}

func CreateTrackMap() Panel {
	return &trackMap{
		mapStore:        CreateTrackMapStore(),
		driverColors:    map[int]color.Color{},
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
	t.driverColors = map[int]color.Color{}
	t.driverNames = map[int]string{}

	t.mapStore.SelectTrack(dataSrc.Track(), dataSrc.SessionStart().Year())
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

	widthf, heightf := giu.CalcTextSize("ABC")
	textHeight := int(heightf/2 + 0.5)
	sideOffset := -(10 + int(widthf+0.5))

	return "Track Map", []giu.Widget{
		giu.Custom(func() {
			canvas := giu.GetCanvas()
			pos := giu.GetCursorScreenPos()

			// Background
			canvas.AddRectFilled(pos.Add(image.Pt(0, 0)), pos.Add(image.Pt(width-15, height-15)), colornames.Black, 0, 0)

			displayWidth := width - 25

			available, scaling, xOffset, yOffset := t.mapStore.MapAvailable(displayWidth, height-25)

			if available {
				t.mapStore.DrawMap(canvas, pos, displayWidth)

				for _, car := range cars {
					x := int(car.X)
					y := int(car.Y)

					// Invert x
					x = displayWidth - x

					x = int((float64(x) / scaling) + 0.5)
					y = int((float64(y) / scaling) + 0.5)

					x += xOffset
					y += yOffset

					driverColor, exists := t.driverColors[car.DriverNumber]
					if !exists {
						driverColor = colornames.White
					}

					canvas.AddCircleFilled(image.Pt(pos.X+x, pos.Y+y), 5.0, driverColor)
					canvas.AddText(image.Pt(pos.X+x+sideOffset, pos.Y+y-textHeight), driverColor, t.driverNames[car.DriverNumber])
				}
			} else {
				textWidth, _ := giu.CalcTextSize("Building Map...")
				offset := int(textWidth / 2)
				canvas.AddText(pos.Add(image.Pt((width/2)-offset, height/2)), colornames.Yellow, "Building Map...")
			}
		}),
	}
}
