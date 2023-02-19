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

package webTimingView

import (
	"context"
	"f1gopher/ui/panel"
	"fmt"
	"github.com/AllenDang/giu"
	"github.com/charmbracelet/lipgloss"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/gorilla/mux"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const timeWidth = 11

type WebTiming struct {
	shutdownWg   sync.WaitGroup
	ctx          context.Context
	started      bool
	stopTickChan chan bool

	servers      []string
	listeners    []*http.Server
	redrawTicker *time.Ticker

	dataSrc f1gopherlib.F1GopherLib

	data     map[int]Messages.Timing
	dataLock sync.Mutex

	event     Messages.Event
	eventLock sync.Mutex

	rcMessages     []Messages.RaceControlMessage
	rcMessagesLock sync.Mutex

	weather     Messages.Weather
	weatherLock sync.Mutex

	eventTime     time.Time
	remainingTime time.Duration

	fastestSector1        time.Duration
	fastestSector2        time.Duration
	fastestSector3        time.Duration
	theoreticalFastestLap time.Duration
	previousSessionActive Messages.SessionState
	fastestSpeedTrap      int

	gapToInfront bool
	raceSession  bool

	html string
}

func CreateWebTimingView(
	shutdownWg sync.WaitGroup,
	ctx context.Context,
	servers []string) *WebTiming {

	web := WebTiming{
		shutdownWg:   shutdownWg,
		ctx:          ctx,
		servers:      servers,
		data:         map[int]Messages.Timing{},
		started:      false,
		stopTickChan: make(chan bool),
	}
	return &web
}

func (w *WebTiming) ProcessDrivers(data Messages.Drivers)     {}
func (w *WebTiming) ProcessRadio(data Messages.Radio)         {}
func (w *WebTiming) ProcessLocation(data Messages.Location)   {}
func (w *WebTiming) ProcessTelemetry(data Messages.Telemetry) {}
func (w *WebTiming) Close()                                   {}

func (w *WebTiming) Type() panel.Type { return panel.WebTiming }

func (w *WebTiming) Start() {
	// If we are already running do nothing
	if w.started {
		return
	}

	w.started = true

	w.runWebServer()

	w.shutdownWg.Add(1)

	// Trigger a refresh every second of the html content
	w.redrawTicker = time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				w.Stop()
				return
			case <-w.redrawTicker.C:
				w.updateHTML()
			case <-w.stopTickChan:
				return
			}
		}
	}()
}

func (w *WebTiming) Pause() {
	w.dataSrc = nil
}

func (w *WebTiming) Stop() {
	// If not running do nothing
	if !w.started {
		return
	}

	// tell the ticker routine to stop
	w.stopTickChan <- true

	w.redrawTicker.Stop()
	w.redrawTicker = nil

	// Shutdown listeners
	for x := range w.listeners {
		w.listeners[x].Close()
	}
	w.listeners = nil
	w.shutdownWg.Done()

	w.started = false
}

func (w *WebTiming) Init(dataSrc f1gopherlib.F1GopherLib, config panel.PanelConfig) {
	w.dataSrc = dataSrc
	w.raceSession = dataSrc.Session() == Messages.RaceSession || dataSrc.Session() == Messages.SprintSession
	w.gapToInfront = w.raceSession
}

func (w *WebTiming) ProcessTiming(data Messages.Timing) {
	w.dataLock.Lock()
	w.data[data.Number] = data
	w.dataLock.Unlock()
}

func (w *WebTiming) ProcessEventTime(data Messages.EventTime) {
	w.eventTime = data.Timestamp
	w.remainingTime = data.Remaining
}

func (w *WebTiming) ProcessEvent(data Messages.Event) {
	w.eventLock.Lock()
	w.event = data
	w.eventLock.Unlock()
}

func (w *WebTiming) ProcessRaceControlMessages(data Messages.RaceControlMessage) {
	w.rcMessagesLock.Lock()
	w.rcMessages = append(w.rcMessages, data)
	w.rcMessagesLock.Unlock()
}

func (w *WebTiming) ProcessWeather(data Messages.Weather) {
	w.weatherLock.Lock()
	w.weather = data
	w.weatherLock.Unlock()
}

func (w *WebTiming) Draw(width int, height int) (widgets []giu.Widget) {
	return nil
}

func (w *WebTiming) runWebServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", func(writer http.ResponseWriter, r *http.Request) {
		abc := `<html>
<title>F1Gopher</title>
<head>    
	<meta charset="utf-8">
</head>
<script language="javascript">	
	async function subscribe() {
  		let response = await fetch("/data");
	
		let message = await response.text();
		document.getElementById("display").innerHTML = message;
		
		await new Promise(resolve => setTimeout(resolve, 1000));
		await subscribe();
  	}

	subscribe();

</script>
<body style="background-color:black; color:white">
	<div>
		<pre id="display"></pre>
	</div>
</body>
</html>`

		writer.Write([]byte(abc))
	})

	router.HandleFunc("/data", func(writer http.ResponseWriter, r *http.Request) {
		writer.Write([]byte(w.html))
	})

	for x := range w.servers {

		srv := &http.Server{
			Handler:      router,
			Addr:         w.servers[x],
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		}

		w.listeners = append(w.listeners, srv)

		go srv.ListenAndServe()
	}
}

func (w *WebTiming) updateHTML() {
	// If no data connection do nothing
	if w.dataSrc == nil {
		w.html = "No data source selected."
		return
	}

	drivers := make([]Messages.Timing, 0)
	w.dataLock.Lock()
	for _, a := range w.data {
		drivers = append(drivers, a)
	}
	w.dataLock.Unlock()

	// Sort drivers into their position order
	sort.Slice(drivers, func(i, j int) bool {
		return drivers[i].Position < drivers[j].Position
	})

	hour := int(w.remainingTime.Seconds() / 3600)
	minute := int(w.remainingTime.Seconds()/60) % 60
	second := int(w.remainingTime.Seconds()) % 60
	remaining := fmt.Sprintf("%d:%02d:%02d", hour, minute, second)
	segmentCount := w.event.TotalSegments
	if segmentCount == 0 {
		segmentCount = len("Segment")
	}

	var table, separator string
	if w.raceSession {
		table, separator = w.raceDisplay(segmentCount, remaining, drivers)
	} else {
		table, separator = w.practiceQualiDisplay(segmentCount, remaining, drivers)
	}

	table += separator + "\n"
	trackStatus := "Track Status: |"
	for x := 0; x < segmentCount; x++ {
		switch w.event.SegmentFlags[x] {
		case Messages.GreenFlag:
			trackStatus += "<font color=\"#00FF00\">&#x25a0;</font>"
		case Messages.YellowFlag:
			trackStatus += "<font color=\"#FFFF00\">&#x25a0;</font>"
		case Messages.DoubleYellowFlag:
			trackStatus += "<font color=\"#FBFF00\">&#x25a0;</font>"
		case Messages.RedFlag:
			trackStatus += "<font color=\"#FF0000\">&#x25a0;</font>"
		}
		if x == w.event.Sector1Segments-1 || x == w.event.Sector1Segments+w.event.Sector2Segments-1 {
			trackStatus += "|"
		}
	}
	if w.dataSrc.Session() == Messages.RaceSession || w.dataSrc.Session() == Messages.SprintSession {
		trackStatus += fmt.Sprintf("|                   |%s|%s|%s|%s|                        |%s|",
			fmt.Sprintf("<font color=\"#D500D5\">%s</font>", lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(fmtDuration(w.fastestSector1))),
			fmt.Sprintf("<font color=\"#D500D5\">%s</font>", lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(fmtDuration(w.fastestSector2))),
			fmt.Sprintf("<font color=\"#D500D5\">%s</font>", lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(fmtDuration(w.fastestSector3))),
			fmt.Sprintf("<font color=\"#D500D5\">%s</font>", lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(fmtDuration(w.theoreticalFastestLap))),
			fmt.Sprintf("<font color=\"#D500D5\">%s</font>", lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Render(fmt.Sprintf("%d", w.fastestSpeedTrap))))
	} else {
		trackStatus += fmt.Sprintf("|                       |%s|%s|%s|%s|                |%s|",
			fmt.Sprintf("<font color=\"#D500D5\">%s</font>", lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(w.fastestSector1))),
			fmt.Sprintf("<font color=\"#D500D5\">%s</font>", lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(w.fastestSector2))),
			fmt.Sprintf("<font color=\"#D500D5\">%s</font>", lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(w.fastestSector3))),
			fmt.Sprintf("<font color=\"#D500D5\">%s</font>", lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(w.theoreticalFastestLap))),
			fmt.Sprintf("<font color=\"#D500D5\">%s</font>", lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Render(fmt.Sprintf("%d", w.fastestSpeedTrap))))
	}

	table += trackStatus + "\n"

	table += separator + "\n"
	w.rcMessagesLock.Lock()
	if len(w.rcMessages) > 0 {
		for x := len(w.rcMessages) - 1; x >= 0 && x >= len(w.rcMessages)-19; x-- {
			lastMessage := w.rcMessages[x]
			prefix := ""

			switch lastMessage.Flag {
			case Messages.ChequeredFlag:
				prefix = "🏁 "
			case Messages.GreenFlag:
				if strings.HasPrefix(lastMessage.Msg, "GREEN LIGHT") {
					prefix = "<font color=\"#00FF00\">&#11044; </font>"
				} else {
					prefix = "<font color=\"#00FF00\">&#x2691; </font>"
				}
			case Messages.YellowFlag:
				prefix = "<font color=\"#FFFF00\">&#x2691; </font>"
			case Messages.DoubleYellowFlag:
				prefix = "<font color=\"#FFFF00\">&#x2691;&#x2691; </font>"
			case Messages.BlueFlag:
				prefix = "<font color=\"#0000FF\">&#x2691; </font>"
			case Messages.RedFlag:
				if strings.HasPrefix(lastMessage.Msg, "RED LIGHT") {
					prefix = "<font color=\"#FF0000\">&#11044; </font>"
				} else {
					prefix = "<font color=\"#FF0000\">&#x2691; </font>"
				}
			case Messages.BlackAndWhite:
				prefix = "<font color=\"#000000\">&#x2691;</font>" + "<font color=\"#FFFFFF\">&#x2691; </font>"
			}

			table += fmt.Sprintf("%s - %s%s\n", lastMessage.Timestamp.In(w.dataSrc.CircuitTimezone()).Format("02-01-2006 15:04:05"), prefix, lastMessage.Msg)
		}
	}
	w.rcMessagesLock.Unlock()

	status := separator + "\n"

	w.weatherLock.Lock()
	status += fmt.Sprintf("Air Temp: %.2f°C, Track Temp: %.2f°C", w.weather.AirTemp, w.weather.TrackTemp)
	if w.weather.Rainfall {
		status += ", <font color=\"#009DD3\">Raining</font>"
	}
	w.weatherLock.Unlock()

	// If it is a race and the session hasn't started yet (remaining time count down hasn't started) then
	// display a count down to the start of the session
	if (w.dataSrc.Session() == Messages.RaceSession || w.dataSrc.Session() == Messages.SprintSession) && w.event.Status == Messages.UnknownState {
		status += fmt.Sprintf(", <font color=\"#00FF00\">Session Starts in: %s</font>", fmtCountdown(w.dataSrc.SessionStart().Sub(w.eventTime)))
	}

	table += status

	w.html = table
}

func (w *WebTiming) practiceQualiDisplay(segmentCount int, remaining string, v []Messages.Timing) (table string, separator string) {

	separator = "------------------------------------------------------------------------------------------------------------------------------------------------------"

	title := fmt.Sprintf("%s: %v, Track Time: %v, Status: %s, DRS: %s, Remaining: %s %s\n",
		w.dataSrc.Name(),
		w.event.Type.String(),
		w.eventTime.In(w.dataSrc.CircuitTimezone()).Format("2006-01-02 15:04:05"),
		fmt.Sprintf("<font color=\"%s\">%s</font>", sessionStatusColor(w.event.Status), w.event.Status.String()),
		w.event.DRSEnabled.String(),
		remaining,
		fmt.Sprintf("<font color=\"%s\">&#x2691</font>", trackStatusColor(w.event.TrackStatus)))

	header := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
		lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render("Pos"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Padding(0, 1, 0, 1).Render("Driver"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(segmentCount+2).Padding(0, 1, 0, 1).Render("Segment"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render("Fastest"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render("Gap"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render("S1"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render("S2"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render("S3"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render("Last Lap"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(10).Padding(0, 1, 0, 1).Render("Tire"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render("Lap"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Render("Speed"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(10).Padding(0, 1, 0, 1).Render("Location"))

	table = title + header + "\n" + separator + "\n"

	outBackground := "#4545E4"
	dropZoneBackground := "#53544E"

	for x, driver := range v {
		speedTrap := ""
		if driver.SpeedTrap > 0 {
			speedTrap = fmt.Sprintf("%d", driver.SpeedTrap)
		}

		gap := driver.TimeDiffToFastest
		if w.gapToInfront {
			gap = driver.TimeDiffToPositionAhead
		}

		segments := ""
		for x := 0; x < segmentCount; x++ {
			switch driver.Segment[x] {
			case Messages.None:
				segments += " "
			default:
				segments += fmt.Sprintf("<font color=\"%s\">&#x25a0;</font>", segmentColor(driver.Segment[x]))
			}

			if x == w.event.Sector1Segments-1 || x == w.event.Sector1Segments+w.event.Sector2Segments-1 {
				segments += "|"
			}
		}

		var row string
		if !driver.KnockedOutOfQualifying {

			if w.event.Type == Messages.Qualifying1 && x >= 15 ||
				w.event.Type == Messages.Qualifying2 && x >= 10 {

				row = fmt.Sprintf("<pr style=\"background-color: %s\">%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s</pr>",
					dropZoneBackground,
					lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render(fmt.Sprintf("%d", driver.Position)),
					fmt.Sprintf("<font color=\"%s\">%s</font>", driver.HexColor, lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Padding(0, 1, 0, 1).Render(driver.ShortName)),
					segments,
					lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.FastestLap)),
					lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(gap)),
					fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.Sector1PersonalFastest, driver.Sector1OverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.Sector1))),
					fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.Sector2PersonalFastest, driver.Sector2OverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.Sector2))),
					fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.Sector3PersonalFastest, driver.Sector3OverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.Sector3))),
					fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.LastLapPersonalFastest, driver.LastLapOverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.LastLap))),
					fmt.Sprintf("<font color=\"%s\">%s</font>", tireColor(driver.Tire), lipgloss.NewStyle().Align(lipgloss.Center).Width(10).Padding(0, 1, 0, 1).Render(driver.Tire.String())),
					lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render(fmt.Sprintf("%d", driver.LapsOnTire)),
					fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.SpeedTrapPersonalFastest, driver.SpeedTrapOverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Render(speedTrap)),
					fmt.Sprintf("<font color=\"%s\">%s</font>", locationColor(driver.Location), lipgloss.NewStyle().Align(lipgloss.Center).Width(10).Padding(0, 1, 0, 1).Render(driver.Location.String())))

			} else {
				row = fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
					lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render(fmt.Sprintf("%d", driver.Position)),
					fmt.Sprintf("<font color=\"%s\">%s</font>", driver.HexColor, lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Padding(0, 1, 0, 1).Render(driver.ShortName)),
					segments,
					lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.FastestLap)),
					lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(gap)),
					fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.Sector1PersonalFastest, driver.Sector1OverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.Sector1))),
					fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.Sector2PersonalFastest, driver.Sector2OverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.Sector2))),
					fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.Sector3PersonalFastest, driver.Sector3OverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.Sector3))),
					fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.LastLapPersonalFastest, driver.LastLapOverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.LastLap))),
					fmt.Sprintf("<font color=\"%s\">%s</font>", tireColor(driver.Tire), lipgloss.NewStyle().Align(lipgloss.Center).Width(10).Padding(0, 1, 0, 1).Render(driver.Tire.String())),
					lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render(fmt.Sprintf("%d", driver.LapsOnTire)),
					fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.SpeedTrapPersonalFastest, driver.SpeedTrapOverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Render(speedTrap)),
					fmt.Sprintf("<font color=\"%s\">%s</font>", locationColor(driver.Location), lipgloss.NewStyle().Align(lipgloss.Center).Width(10).Padding(0, 1, 0, 1).Render(driver.Location.String())))
			}

			if driver.ChequeredFlag {
				row = row + " 🏁"
			}

		} else {

			row = fmt.Sprintf("<pr style=\"background-color: %s\">%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s</pr>",
				outBackground,
				lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render(fmt.Sprintf("%d", driver.Position)),
				fmt.Sprintf("<font color=\"%s\">%s</font>", driver.Color, lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Padding(0, 1, 0, 1).Render(driver.ShortName)),
				lipgloss.NewStyle().Align(lipgloss.Left).Width(segmentCount+2).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.FastestLap)),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(10).Padding(0, 1, 0, 1).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(10).Padding(0, 1, 0, 1).Render("Out"))
		}

		table += row + "\n"
	}

	return table, separator
}

func (w *WebTiming) raceDisplay(segmentCount int, remaining string, v []Messages.Timing) (table string, separator string) {

	separator = "---------------------------------------------------------------------------------------------------------------------------------------------"

	title := fmt.Sprintf("%s: %v, Track Time: %v, Status: %s, DRS: %v, Safety Car: %s, Lap: %d/%d, Remaining: %s %s\n",
		w.dataSrc.Name(),
		w.event.Type.String(),
		w.eventTime.In(w.dataSrc.CircuitTimezone()).Format("2006-01-02 15:04:05"),
		fmt.Sprintf("<font color=\"%s\">%s</font>", sessionStatusColor(w.event.Status), w.event.Status.String()),
		w.event.DRSEnabled.String(),
		fmt.Sprintf("<font color=\"%s\">%s</font>", safetyCarFormat(w.event.SafetyCar), w.event.SafetyCar),
		w.event.CurrentLap,
		w.event.TotalLaps,
		remaining,
		fmt.Sprintf("<font color=\"%s\">&#x2691</font>", trackStatusColor(w.event.TrackStatus)))

	header := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
		lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render("Pos"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Padding(0, 1, 0, 1).Render("Driver"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(segmentCount+2).Padding(0, 1, 0, 1).Render("Segment"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render("Fastest"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render("Gap"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render("S1"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render("S2"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render("S3"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render("Last Lap"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(6).Render("DRS"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Render("Tire"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(3).Render("Lap"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(4).Render("Pits"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Render("Speed"),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Render("Location"))

	table = title + header + "\n" + separator + "\n"

	for _, driver := range v {
		if driver.Location == Messages.Stopped {
			row := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
				lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render(fmt.Sprintf("%d", driver.Position)),
				fmt.Sprintf("<font color=\"%s\">%s</font>", driver.HexColor, lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Padding(0, 1, 0, 1).Render(driver.ShortName)),
				lipgloss.NewStyle().Align(lipgloss.Left).Width(segmentCount+2).Render(""),
				fmt.Sprintf("<font color=\"%s\">%s</font>", fastestLapColor(driver.OverallFastestLap), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth).Padding(0, 1, 0, 1).Render(fmtDuration(driver.FastestLap))),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(6).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(3).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(4).Render(""),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Render(""),
				fmt.Sprintf("<font color=\"%s\">%s</font>", locationColor(driver.Location), lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Render(driver.Location.String())))
			table += row + "\n"
			continue
		}

		speedTrap := ""
		if driver.SpeedTrap > 0 {
			speedTrap = fmt.Sprintf("%d", driver.SpeedTrap)
		}

		gap := driver.GapToLeader
		if w.gapToInfront {
			gap = driver.TimeDiffToPositionAhead
		}
		drs := "Closed"
		if driver.DRSOpen {
			drs = "Open"
		}

		segments := ""
		for x := 0; x < segmentCount; x++ {
			switch driver.Segment[x] {
			case Messages.None:
				segments += " "
			default:
				segments += fmt.Sprintf("<font color=\"%s\">&#x25a0;</font>", segmentColor(driver.Segment[x]))
			}

			if x == w.event.Sector1Segments-1 || x == w.event.Sector1Segments+w.event.Sector2Segments-1 {
				segments += "|"
			}
		}

		drsColor := lipgloss.Color("#FFFFFF")
		if driver.TimeDiffToPositionAhead > 0 && driver.TimeDiffToPositionAhead < time.Second {
			drsColor = "#00FF00"
		}

		row := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
			lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Padding(0, 1, 0, 1).Render(fmt.Sprintf("%d", driver.Position)),
			fmt.Sprintf("<font color=\"%s\">%s</font>", driver.HexColor, lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Padding(0, 1, 0, 1).Render(driver.ShortName)),
			segments,
			fmt.Sprintf("<font color=\"%s\">%s</font>", fastestLapColor(driver.OverallFastestLap), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(fmtDuration(driver.FastestLap))),
			lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(fmtDuration(gap)),
			fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.Sector1PersonalFastest, driver.Sector1OverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(fmtDuration(driver.Sector1))),
			fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.Sector2PersonalFastest, driver.Sector2OverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(fmtDuration(driver.Sector2))),
			fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.Sector3PersonalFastest, driver.Sector3OverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(fmtDuration(driver.Sector3))),
			fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.LastLapPersonalFastest, driver.LastLapOverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(timeWidth-2).Render(fmtDuration(driver.LastLap))),
			fmt.Sprintf("<font color=\"%s\">%s</font>", drsColor, lipgloss.NewStyle().Align(lipgloss.Center).Width(6).Render(drs)),
			fmt.Sprintf("<font color=\"%s\">%s</font>", tireColor(driver.Tire), lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Render(driver.Tire.String())),
			lipgloss.NewStyle().Align(lipgloss.Center).Width(3).Render(fmt.Sprintf("%d", driver.LapsOnTire)),
			lipgloss.NewStyle().Align(lipgloss.Center).Width(4).Render(fmt.Sprintf("%d", driver.Pitstops)),
			fmt.Sprintf("<font color=\"%s\">%s</font>", timeColor(driver.SpeedTrapPersonalFastest, driver.SpeedTrapOverallFastest), lipgloss.NewStyle().Align(lipgloss.Center).Width(5).Render(speedTrap)),
			fmt.Sprintf("<font color=\"%s\">%s</font>", locationColor(driver.Location), lipgloss.NewStyle().Align(lipgloss.Center).Width(8).Render(driver.Location.String())))

		if driver.ChequeredFlag {
			row = row + " 🏁"
		}

		table += row + "\n"
	}

	return table, separator
}
