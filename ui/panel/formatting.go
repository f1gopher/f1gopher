package panel

import (
	"fmt"
	"github.com/f1gopher/f1gopherlib/Messages"
	"golang.org/x/image/colornames"
	"image/color"
	"time"
)

func fmtDuration(d time.Duration) string {
	milliseconds := d.Milliseconds()

	if milliseconds == 0 {
		return ""
	}

	minutes := milliseconds / (1000 * 60)
	milliseconds -= minutes * 60 * 1000
	seconds := milliseconds / 1000
	milliseconds -= seconds * 1000

	isNegative := false
	if minutes < 0 {
		minutes = -minutes
		isNegative = true
	}
	if seconds < 0 {
		seconds = -seconds
		isNegative = true
	}
	if milliseconds < 0 {
		milliseconds = -milliseconds
		isNegative = true
	}

	if isNegative {
		return fmt.Sprintf("-%02d:%02d.%03d", minutes, seconds, milliseconds)
	}

	// If no minutes then don't display zero but pad with spaces for display alignment
	if minutes == 0 {
		return fmt.Sprintf("   %02d.%03d", seconds, milliseconds)
	}

	return fmt.Sprintf("%02d:%02d.%03d", minutes, seconds, milliseconds)
}

func timeColor(personalFastest bool, overallFastest bool) color.Color {

	if overallFastest {
		return colornames.Purple
	} else if personalFastest {
		return colornames.Green
	} else {
		return colornames.Yellow
	}
}

func segmentColor(segmentType Messages.SegmentType) color.Color {
	switch segmentType {
	case Messages.YellowSegment:
		return colornames.Yellow
	case Messages.GreenSegment:
		return colornames.Green
	case Messages.InvalidSegment:
		return colornames.Black
	case Messages.PurpleSegment:
		return color.RGBA{R: 213, G: 0, B: 213, A: 255}
	case Messages.RedSegment:
		return colornames.Red
	case Messages.PitlaneSegment:
		return colornames.White
	case Messages.Mystery, Messages.Mystery2, Messages.Mystery3:
		return colornames.Blue
	default:
		return colornames.White
	}
}

func fastestLapColor(overallFastest bool) color.Color {

	if overallFastest {
		return colornames.Purple
	} else {
		// TODO - use default text color
		return colornames.White
	}
}

func tireColor(tire Messages.TireType) color.Color {
	switch tire {
	case Messages.Soft:
		return colornames.Red
	case Messages.Medium:
		return colornames.Yellow
	case Messages.Hard:
		return colornames.White
	case Messages.Intermediate:
		return color.RGBA{R: 37, G: 150, B: 150, A: 255}
	case Messages.Wet:
		return color.RGBA{R: 0, G: 169, B: 255, A: 255}
	default:
		return color.RGBA{R: 105, G: 23, B: 174, A: 255}
	}
}

func locationColor(location Messages.CarLocation) color.Color {
	switch location {
	case Messages.Pitlane, Messages.PitOut, Messages.NoLocation:
		return colornames.White
	case Messages.OnTrack, Messages.OutLap:
		return colornames.Green
	case Messages.Stopped, Messages.OutOfRace:
		return colornames.Red
	default:
		panic("Unhandled location color: " + location.String())
	}
}

func sessionStatusColor(state Messages.SessionState) color.Color {
	switch state {
	case Messages.UnknownState, Messages.Inactive, Messages.Finished, Messages.Finalised, Messages.Ended:
		return colornames.White
	case Messages.Started:
		return colornames.Green
	case Messages.Aborted:
		return colornames.Red
	default:
		panic("Unhandled session status color: " + state.String())
	}
}

func trackStatusColor(state Messages.FlagState) color.Color {
	switch state {
	case Messages.GreenFlag:
		return colornames.Green
	case Messages.YellowFlag, Messages.DoubleYellowFlag:
		return colornames.Yellow
	case Messages.RedFlag:
		return colornames.Red
	case Messages.ChequeredFlag:
		return colornames.White
	case Messages.NoFlag:
		return colornames.White
	default:
		panic("Unhandled track status color: " + state.String())
	}
}

func safetyCarFormat(state Messages.TrackState) color.Color {

	var color = colornames.Green

	switch state {
	case Messages.VirtualSafetyCar, Messages.VirtualSafetyCarEnding:
		color = colornames.Yellow
	case Messages.SafetyCar, Messages.SafetyCarEnding:
		color = colornames.Red
	}

	return color
}
