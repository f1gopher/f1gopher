// F1Gopher - Copyright (C) 2025 f1gopher
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
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
	"sync"
	"time"

	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/ungerik/go-cairo"
	"golang.org/x/image/colornames"
)

type circleMapInfo struct {
	color color.RGBA
	name  string

	gapToLeader  time.Duration
	displayAngle float64
	hasStopped   bool
}

type circleMap struct {
	driverData          map[int]*circleMapInfo
	driverPositionsLock sync.Mutex
	sessionStarted      bool

	trackTexture       *giu.Texture
	trackTextureWidth  float32
	trackTextureHeight float32
	mapGc              *cairo.Surface
	currentWidth       int
	currentHeight      int
}

func CreateCircleMap() Panel {
	panel := &circleMap{
		driverData: map[int]*circleMapInfo{},
	}

	return panel
}

func (c *circleMap) ProcessEventTime(data Messages.EventTime)                    {}
func (c *circleMap) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (c *circleMap) ProcessWeather(data Messages.Weather)                        {}
func (c *circleMap) ProcessRadio(data Messages.Radio)                            {}
func (c *circleMap) ProcessLocation(data Messages.Location)                      {}
func (c *circleMap) ProcessTelemetry(data Messages.Telemetry)                    {}
func (c *circleMap) Close()                                                      {}

func (c *circleMap) Type() Type { return CircleMap }

func (c *circleMap) Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig) {
	c.driverData = map[int]*circleMapInfo{}
	if c.mapGc != nil {
		c.mapGc.Destroy()
		c.mapGc = nil
	}
	c.currentWidth = 0
	c.currentHeight = 0
	c.sessionStarted = false
}

func (c *circleMap) ProcessDrivers(data Messages.Drivers) {
	for x := range data.Drivers {
		c.driverData[data.Drivers[x].Number] = &circleMapInfo{
			color:        data.Drivers[x].Color,
			name:         data.Drivers[x].ShortName,
			gapToLeader:  time.Duration(0),
			displayAngle: 0.0,
			hasStopped:   false,
		}
	}
}

func (c *circleMap) ProcessEvent(data Messages.Event) {
	if data.Status == Messages.Started {
		c.sessionStarted = true
	}
}

func (c *circleMap) ProcessTiming(data Messages.Timing) {
	c.driverPositionsLock.Lock()
	defer c.driverPositionsLock.Unlock()

	driverData := c.driverData[data.Number]
	if driverData == nil {
		return
	}
	driverData.gapToLeader = data.GapToLeader
	// 1 minute or 60 second circle
	angleDegrees := (360.0 / 60.0) * driverData.gapToLeader.Seconds()
	driverData.displayAngle = 0.01745329 * angleDegrees
	driverData.hasStopped = data.Location == Messages.OutOfRace || data.Location == Messages.Stopped

}

func (c *circleMap) Draw(width int, height int) []giu.Widget {
	c.redraw(width, height)

	if c.trackTexture != nil {
		return []giu.Widget{
			giu.Image(c.trackTexture).Size(c.trackTextureWidth, c.trackTextureHeight),
		}
	}

	return []giu.Widget{}
}

func (c *circleMap) redraw(width int, height int) {
	displayWidth := width - 16
	displayHeight := height - 16

	if c.mapGc == nil || displayWidth != c.currentWidth || displayHeight != c.currentHeight {
		if c.mapGc != nil {
			c.mapGc.Destroy()
		}
		c.mapGc = cairo.NewSurface(cairo.FORMAT_ARGB32, displayWidth, displayHeight)
		c.mapGc.SelectFontFace("sans-serif", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
		c.mapGc.SetFontSize(10.0)
		c.currentWidth = width
		c.currentHeight = height
		c.trackTextureWidth = float32(displayWidth)
		c.trackTextureHeight = float32(displayHeight)
	}

	c.mapGc.Translate(float64(displayWidth)/2, float64(displayHeight)/2)

	if !c.sessionStarted {
		color := colornames.White
		c.mapGc.SetSourceRGBA(float64(color.R)/255.0, float64(color.G)/255.0, float64(color.B)/255.0, 1.0)
		c.mapGc.Translate(-100, 0)
		c.mapGc.SetFontSize(20.0)
		c.mapGc.ShowText("Waiting for the race to start...")
		c.mapGc.Stroke()
	} else {
		drawColor := colornames.White
		c.mapGc.SetSourceRGBA(float64(drawColor.R)/255.0, float64(drawColor.G)/255.0, float64(drawColor.B)/255.0, 1.0)

		radius := (float64(displayHeight) - 30.0) / 2.0

		c.mapGc.Arc(0.0, 0.0, radius, 0, 2*math.Pi)
		c.mapGc.StrokePreserve()
		c.mapGc.Stroke()
		outside := true

		for _, driver := range c.driverData {
			if driver.hasStopped || driver.gapToLeader > time.Second*55 {
				continue
			}

			drawColor = driver.color
			c.mapGc.SetSourceRGBA(float64(drawColor.R)/255.0, float64(drawColor.G)/255.0, float64(drawColor.B)/255.0, 1.0)

			c.mapGc.MoveTo(c.pointForAngle(-driver.displayAngle, radius-10))
			c.mapGc.LineTo(c.pointForAngle(-driver.displayAngle, radius+10))

			nameOffset := radius - 30
			c.mapGc.MoveTo(c.pointForAngle(-driver.displayAngle, nameOffset))
			c.mapGc.ShowText(driver.name)

			c.mapGc.Stroke()

			outside = !outside
		}
	}
	c.mapGc.Flush()

	trueImg := c.mapGc.GetImage()
	rgba := image.NewRGBA(trueImg.Bounds())
	draw.Draw(rgba, trueImg.Bounds(), trueImg, image.Pt(0, 0), draw.Src)
	giu.NewTextureFromRgba(rgba, func(tex *giu.Texture) {
		c.trackTexture = tex
	})
}

func (c *circleMap) pointForAngle(angle float64, radius float64) (x, y float64) {
	x = radius * math.Sin(angle)
	y = radius * math.Cos(angle)
	return x, -y
}
