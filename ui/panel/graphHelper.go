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
	pixelGapForOne float64,
	majorTickIncrement float64) {

	// Y Axis line
	dc.MoveTo(xPos, topY)
	dc.LineTo(xPos, bottomY)

	valueXPos := xPos - 40
	valueYOffset := 3.0
	majorTickStartX := xPos - 8
	minorTickStartX := xPos - 4

	gapForMajorTick := pixelGapForOne * majorTickIncrement

	// Major ticks with values from the reference value up to the top
	currentTickValue := referenceValue
	yPos := referenceYPos
	for yPos > topY {
		dc.MoveTo(valueXPos, yPos+valueYOffset)
		dc.ShowText(fmt.Sprintf("%5.0f", currentTickValue))

		dc.MoveTo(majorTickStartX, yPos)
		dc.LineTo(xPos, yPos)

		yPos -= gapForMajorTick
		currentTickValue += majorTickIncrement
	}

	// Major ticks with value from the reference value down to the bottom
	currentTickValue = referenceValue - majorTickIncrement
	yPos = referenceYPos + gapForMajorTick
	for yPos < bottomY {
		dc.MoveTo(valueXPos, yPos+valueYOffset)
		dc.ShowText(fmt.Sprintf("%5.0f", currentTickValue))

		dc.MoveTo(majorTickStartX, yPos)
		dc.LineTo(xPos, yPos)

		yPos += gapForMajorTick
		currentTickValue -= majorTickIncrement
	}

	// Find the most suitable number of minor ticks to show. Default to majorTickIncrement/10 but
	// if the gap is too small then do majorTickIncrement/2 instead
	pixelGapForMinorTick := gapForMajorTick / 10.0
	valueIncrement := majorTickIncrement / 10.0
	if pixelGapForMinorTick < 10.0 {
		pixelGapForMinorTick = gapForMajorTick / 2.0
		valueIncrement = majorTickIncrement / 2.0
	}

	// Minor ticks start from just above the reference value up to the top
	currentTickValue = referenceValue + valueIncrement
	yPos = referenceYPos - pixelGapForMinorTick
	for yPos > topY {
		dc.MoveTo(minorTickStartX, yPos)
		dc.LineTo(xPos, yPos)

		yPos -= pixelGapForMinorTick
		currentTickValue += valueIncrement
	}

	// Minor ticks start from just below the reference value down to the bottom
	currentTickValue = referenceValue - valueIncrement
	yPos = referenceYPos + pixelGapForMinorTick
	for yPos < bottomY {
		dc.MoveTo(minorTickStartX, yPos)
		dc.LineTo(xPos, yPos)

		yPos += pixelGapForMinorTick
		currentTickValue -= valueIncrement
	}

	dc.Stroke()
}
