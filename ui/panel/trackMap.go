package panel

import (
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
)

type trackMap struct {
}

func CreateTrackMap() Panel {
	return &trackMap{}
}

func (t *trackMap) Init(dataSrc f1gopherlib.F1GopherLib)                        {}
func (t *trackMap) Close()                                                      {}
func (t *trackMap) ProcessTiming(data Messages.Timing)                          {}
func (t *trackMap) ProcessEventTime(data Messages.EventTime)                    {}
func (t *trackMap) ProcessEvent(data Messages.Event)                            {}
func (t *trackMap) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (t *trackMap) ProcessWeather(data Messages.Weather)                        {}
func (t *trackMap) ProcessRadio(data Messages.Radio)                            {}

func (t *trackMap) ProcessLocation(data Messages.Location) {

}

func (t *trackMap) Draw() (title string, widgets []giu.Widget) {
	return "Track Map", nil
}
