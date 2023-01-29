package panel

import (
	"fmt"
	"github.com/ungerik/go-cairo"
)

func drawYAxis(
	dc *cairo.Surface,
	xPos float64,
	topY float64,
	bottomY float64,
	referenceYPos float64,
	referenceValue float64,
	pixelGapForOne float64) {

	// Y Axis line
	dc.SetSourceRGB(0.0, 0.0, 1.0)
	dc.MoveTo(xPos, topY)
	dc.LineTo(xPos, bottomY)
	dc.Stroke()

	valueXPos := xPos - 25
	valueYOffset := 3.0
	majorTickStartX := xPos - 10
	minorTickStartX := xPos - 5

	// Major ticks with values from the reference value up to the top
	currentTickValue := referenceValue
	yPos := referenceYPos
	for yPos > topY {
		dc.MoveTo(valueXPos, yPos+valueYOffset)
		dc.ShowText(fmt.Sprintf("%.0f", currentTickValue))

		dc.MoveTo(majorTickStartX, yPos)
		dc.LineTo(xPos, yPos)
		dc.Stroke()

		yPos -= pixelGapForOne
		currentTickValue += 1.0
	}

	// Major ticks with value from the reference value down to the bottom
	currentTickValue = referenceValue - 1.0
	yPos = referenceYPos + pixelGapForOne
	for yPos < bottomY {
		dc.MoveTo(valueXPos, yPos+valueYOffset)
		dc.ShowText(fmt.Sprintf("%.0f", currentTickValue))

		dc.MoveTo(majorTickStartX, yPos)
		dc.LineTo(xPos, yPos)
		dc.Stroke()

		yPos += pixelGapForOne
		currentTickValue -= 1.0
	}

	// Find the most suitable number of minor ticks to show. Default to 0.1 but
	// if the gap is too small then do 0.5 instead
	pixelGapForMinorTick := pixelGapForOne / 10.0
	valueIncrement := 0.1
	if pixelGapForMinorTick < 10.0 {
		pixelGapForMinorTick = pixelGapForOne / 2.0
		valueIncrement = 0.5
	}

	// Minor ticks start from just above the reference value up to the top
	currentTickValue = referenceValue + valueIncrement
	yPos = referenceYPos + pixelGapForMinorTick
	for yPos > topY {
		dc.MoveTo(minorTickStartX, yPos)
		dc.LineTo(xPos, yPos)
		dc.Stroke()

		yPos -= pixelGapForMinorTick
		currentTickValue += valueIncrement
	}

	// Minor ticks start from just below the reference value down to the bottom
	currentTickValue = referenceValue - valueIncrement
	yPos = referenceYPos - pixelGapForMinorTick
	for yPos < bottomY {
		dc.MoveTo(minorTickStartX, yPos)
		dc.LineTo(xPos, yPos)
		dc.Stroke()

		yPos += pixelGapForMinorTick
		currentTickValue -= valueIncrement
	}
}
