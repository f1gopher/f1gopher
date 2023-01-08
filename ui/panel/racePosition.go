package panel

import (
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"image/color"
	"sort"
	"sync"
)

type info struct {
	color     color.RGBA
	number    int
	name      string
	positions []float64
}

type racePosition struct {
	driverData  map[int]*info
	orderedData []*info
	totalLaps   int

	driverPositionsLock sync.Mutex

	lines []giu.PlotWidget
}

func CreateRacePosition() Panel {
	return &racePosition{
		lines:       []giu.PlotWidget{},
		driverData:  map[int]*info{},
		orderedData: []*info{},
		totalLaps:   0,
	}
}

func (r *racePosition) Close()                                                      {}
func (r *racePosition) ProcessEventTime(data Messages.EventTime)                    {}
func (r *racePosition) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (r *racePosition) ProcessWeather(data Messages.Weather)                        {}
func (r *racePosition) ProcessRadio(data Messages.Radio)                            {}
func (r *racePosition) ProcessLocation(data Messages.Location)                      {}

func (r *racePosition) Type() Type { return RacePosition }

func (r *racePosition) Init(dataSrc f1gopherlib.F1GopherLib) {
	// Clear previous session data
	r.lines = []giu.PlotWidget{}
	r.driverData = map[int]*info{}
	r.orderedData = []*info{}
	r.totalLaps = 0
}

func (r *racePosition) ProcessEvent(data Messages.Event) {
	if r.totalLaps == 0 {
		r.totalLaps = data.TotalLaps
	}
}

func (r *racePosition) ProcessTiming(data Messages.Timing) {

	driverInfo, exists := r.driverData[data.Number]
	if !exists {
		r.driverPositionsLock.Lock()
		defer r.driverPositionsLock.Unlock()

		driverInfo = &info{
			color:     data.Color,
			number:    data.Number,
			name:      data.ShortName,
			positions: []float64{float64(data.Position)},
		}

		r.driverData[data.Number] = driverInfo
		r.orderedData = append(r.orderedData, driverInfo)
		sort.Slice(r.orderedData, func(i, j int) bool {
			return r.orderedData[i].positions[0] < r.orderedData[j].positions[0]
		})

		tmpLines := []giu.PlotWidget{}
		for x := range r.orderedData {
			tmpLines = append(tmpLines, giu.PlotLine(r.orderedData[x].name, r.orderedData[x].positions))
		}

		r.lines = tmpLines
	} else {
		count := len(driverInfo.positions)
		if count == data.Lap {
			driverInfo.positions = append(driverInfo.positions, float64(data.Position))

			tmpLines := []giu.PlotWidget{}
			for x := range r.orderedData {
				tmpLines = append(tmpLines, giu.PlotLine(r.orderedData[x].name, r.orderedData[x].positions))
			}

			r.driverPositionsLock.Lock()
			defer r.driverPositionsLock.Unlock()
			r.lines = tmpLines
		}
	}
}

func (r *racePosition) Draw(width int, height int) []giu.Widget {
	return []giu.Widget{
		giu.Plot("Race Position").Plots(r.lines...).
			Size(width-16, height-16).
			AxisLimits(0, float64(r.totalLaps), 0, float64(len(r.driverData)), giu.ConditionAppearing),
	}
}
