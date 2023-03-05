package panel

import (
	"time"
)

type PanelConfig interface {
	PredictedPitstopTime() time.Duration
	SetPredictedPitstopTime(value time.Duration)
}
