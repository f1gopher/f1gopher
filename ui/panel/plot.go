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
	"github.com/AllenDang/giu"
	"github.com/ungerik/go-cairo"
	"image/color"
)

type plot struct {
	plotTexture       *giu.Texture
	plotTextureWidth  float32
	plotTextureHeight float32

	backgroundGc *cairo.Surface
	foregroundGc *cairo.Surface

	currentWidth  int
	currentHeight int
	widget        *giu.ImageWidget

	redrawForeground bool
	redrawBackground bool

	drawBackground func(*cairo.Surface)
	drawForeground func(*cairo.Surface)
}

func createPlot(drawBackground func(*cairo.Surface), drawForeground func(*cairo.Surface)) *plot {
	return &plot{
		drawBackground: drawBackground,
		drawForeground: drawForeground,
	}
}

func (p *plot) reset() {
	p.refreshBackground()

	if p.foregroundGc != nil {
		// Clear the displayed image
		p.foregroundGc.SetSourceRGB(0.0, 0.0, 0.0)
		p.foregroundGc.Rectangle(0, 0, float64(p.currentWidth), float64(p.currentHeight))
		p.foregroundGc.Fill()
		p.foregroundGc.Stroke()
		p.foregroundGc.Flush()

		img := p.foregroundGc.GetImage()
		giu.EnqueueNewTextureFromRgba(img, func(texture *giu.Texture) {
			p.plotTexture = texture
		})

		// Update image widget
		p.widget = giu.Image(p.plotTexture).Size(p.plotTextureWidth, p.plotTextureHeight)
	}
}

func (p *plot) draw(width int, height int) *giu.ImageWidget {

	p.redraw(width, height)

	return p.widget
}

func (p *plot) refreshForeground() {
	p.redrawForeground = true
}

func (p *plot) refreshBackground() {
	p.redrawBackground = true
}

func (p *plot) redraw(width int, height int) {
	sizeChanged := width != p.currentWidth || height != p.currentHeight

	// If the size hasn't changed or a draw isn't required don't do anything
	if !sizeChanged && !p.redrawBackground && !p.redrawForeground {
		return
	}

	if p.backgroundGc == nil || p.foregroundGc == nil || sizeChanged {
		p.backgroundGc = cairo.NewSurface(cairo.FORMAT_ARGB32, width, height)
		p.backgroundGc.SelectFontFace("sans-serif", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
		p.backgroundGc.SetFontSize(10.0)

		p.foregroundGc = cairo.NewSurface(cairo.FORMAT_ARGB32, width, height)
		p.foregroundGc.SelectFontFace("sans-serif", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
		p.foregroundGc.SetFontSize(10.0)

		p.currentWidth = width
		p.currentHeight = height
		p.plotTextureWidth = float32(width)
		p.plotTextureHeight = float32(height)
	}

	backgroundChanged := false

	// Redraw the background if needed
	if p.redrawBackground || sizeChanged {
		p.drawBackground(p.backgroundGc)
		p.backgroundGc.Flush()
		p.redrawBackground = false
		backgroundChanged = true
	}

	// Copy background data to reset foreground
	p.foregroundGc.SetData(p.backgroundGc.GetData())

	// Redraw the foreground
	if p.redrawForeground || backgroundChanged || sizeChanged {
		p.drawForeground(p.foregroundGc)
		p.redrawForeground = false
	}

	// Copy image to display texture
	p.foregroundGc.Flush()
	img := p.foregroundGc.GetImage()
	giu.EnqueueNewTextureFromRgba(img, func(texture *giu.Texture) {
		p.plotTexture = texture
	})

	// Update image widget
	p.widget = giu.Image(p.plotTexture).Size(p.plotTextureWidth, p.plotTextureHeight)
}

func floatColor(color color.RGBA) (float64, float64, float64, float64) {
	return float64(color.R) / 255.0, float64(color.G) / 255.0, float64(color.B) / 255.0, float64(color.A) / 255.0
}
