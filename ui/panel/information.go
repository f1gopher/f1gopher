package panel

import (
	"fmt"
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"sync"
	"time"
)

type information struct {
	exit    func()
	dataSrc f1gopherlib.F1GopherLib

	event         Messages.Event
	eventLock     sync.Mutex
	eventTime     time.Time
	remainingTime time.Duration
}

func CreateInformation(exit func()) Panel {
	return &information{
		exit: exit,
	}
}

func (i *information) ProcessTiming(data Messages.Timing)                          {}
func (i *information) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (i *information) ProcessWeather(data Messages.Weather)                        {}
func (i *information) ProcessRadio(data Messages.Radio)                            {}
func (i *information) ProcessLocation(data Messages.Location)                      {}
func (i *information) Close()                                                      {}

func (i *information) Init(dataSrc f1gopherlib.F1GopherLib) {
	i.dataSrc = dataSrc

	// Clear previous session data
	i.event = Messages.Event{}
	i.remainingTime = 0
}

func (i *information) ProcessEventTime(data Messages.EventTime) {
	i.eventTime = data.Timestamp
	i.remainingTime = data.Remaining
}

func (i *information) ProcessEvent(data Messages.Event) {
	i.eventLock.Lock()
	i.event = data
	i.eventLock.Unlock()
}

func (i *information) Draw() (title string, widgets []giu.Widget) {

	pauseTxt := "Pause"
	if i.dataSrc.IsPaused() {
		pauseTxt = "Resume"
	}

	panelWidgets := []giu.Widget{
		i.infoWidgets(),

		// Temporary time skip controls (TODO - need to hide for live view)
		giu.Row(
			giu.Button("Jump to Start").OnClick(func() {
				i.dataSrc.SkipToSessionStart()
			}),
			giu.Button("Skip Minute").OnClick(func() {
				i.dataSrc.IncrementTime(time.Minute * 1)
			}),
			giu.Button(pauseTxt).OnClick(func() {
				i.dataSrc.TogglePause()
			}),
			giu.Button("Back").OnClick(func() {
				i.exit()
			}),
		),
	}

	return "Information", panelWidgets
}

func (i *information) infoWidgets() *giu.RowWidget {
	hour := int(i.remainingTime.Seconds() / 3600)
	minute := int(i.remainingTime.Seconds()/60) % 60
	second := int(i.remainingTime.Seconds()) % 60
	remaining := fmt.Sprintf("%d:%02d:%02d", hour, minute, second)

	i.eventLock.Lock()
	defer i.eventLock.Unlock()

	widgets := []giu.Widget{
		giu.Label(fmt.Sprintf(
			"%s: %v, Track Time: %v, Status:",
			i.dataSrc.Name(),
			i.event.Type.String(),
			i.eventTime.In(i.dataSrc.CircuitTimezone()).Format("2006-01-02 15:04:05"))),
		giu.Style().SetColor(giu.StyleColorText, sessionStatusColor(i.event.Status)).To(
			giu.Label(i.event.TrackStatus.String())),
	}

	// These are only relevant for a race session
	if i.event.Type == Messages.Race || i.event.Type == Messages.Sprint {
		widgets = append(widgets,
			giu.Label(fmt.Sprintf(", DRS: %v, Safety Car:",
				i.event.DRSEnabled.String())))

		widgets = append(widgets,
			giu.Style().SetColor(giu.StyleColorText, safetyCarFormat(i.event.SafetyCar)).To(
				giu.Label(i.event.SafetyCar.String())))

		widgets = append(widgets,
			giu.Label(fmt.Sprintf(", Lap: %d/%d, Remaining: %s",
				i.event.CurrentLap,
				i.event.TotalLaps,
				remaining)))
	}

	widgets = append(widgets, giu.Style().SetColor(giu.StyleColorText, trackStatusColor(i.event.TrackStatus)).To(
		giu.Label("⚑")))

	return giu.Row(widgets...)
}
