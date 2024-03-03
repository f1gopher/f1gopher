// F1Gopher - Copyright (C) 2022 f1gopher
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

package ui

import (
	"context"
	"f1gopher/ui/panel"
	"sync"

	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
)

type dataView struct {
	ctxShutdown context.CancelFunc
	ctx         context.Context
	closing     bool

	dataSrc f1gopherlib.F1GopherLib

	changeView func(newView screen, info any)

	panels map[panel.Type]panel.Panel

	event     Messages.Event
	eventLock sync.Mutex
	closeWg   sync.WaitGroup

	showTelemetry bool

	layoutFunc func(width int, height int)
}

func createDataView(webView panel.Panel, changeView func(newView screen, info any), isLiveSession bool) dataScreen {
	view := dataView{
		changeView: changeView,
		panels:     map[panel.Type]panel.Panel{},
	}

	view.layoutFunc = view.newLayout

	view.addPanel(panel.CreateInformation(func() { changeView(MainMenu, nil) }, isLiveSession))
	view.addPanel(panel.CreateTiming())
	view.addPanel(panel.CreateRaceControlMessages())
	view.addPanel(panel.CreateWeather())
	view.addPanel(panel.CreateTeamRadio())
	view.addPanel(panel.CreateTrackMap())
	view.addPanel(panel.CreateTelemetry())

	// TODO - only create these for race session so that we don't have them processing data even when not displayed
	view.addPanel(panel.CreateRacePosition())
	view.addPanel(panel.CreateGapperPlot())
	view.addPanel(panel.CreateCatching())

	// Quali only
	view.addPanel(panel.CreateImproving())

	view.addPanel(webView)

	return &view
}

func (d *dataView) toggleTelemetryView() {
	d.showTelemetry = !d.showTelemetry
}

func (d *dataView) addPanel(panel panel.Panel) {
	d.panels[panel.Type()] = panel
}

func (d *dataView) init(dataSrc f1gopherlib.F1GopherLib, config config) {
	d.dataSrc = dataSrc
	d.ctx, d.ctxShutdown = context.WithCancel(context.Background())
	d.closing = false

	// Reset the global pitstop loss time to the currently selected track default
	config.SetPredictedPitstopTime(dataSrc.TimeLostInPitlane())

	for x := range d.panels {
		d.panels[x].Init(dataSrc, &config)
	}

	// Listen for and handle data messages in the background
	go d.processData()
}

func (d *dataView) close() {
	d.closing = true
	d.dataSrc.Close()

	if d.ctxShutdown != nil {
		d.ctxShutdown()
	}

	// Wait for drawing to finish
	d.closeWg.Wait()

	for x := range d.panels {
		d.panels[x].Close()
	}

	// Reset for the next session
	d.event = Messages.Event{}
	d.dataSrc = nil
	d.ctx = nil
	d.ctxShutdown = nil
}

func (d *dataView) draw(width int, height int) {
	d.layoutFunc(width, height)
}

func (d *dataView) newLayout(width int, height int) {
	if d.closing {
		return
	}

	d.closeWg.Add(1)
	defer d.closeWg.Done()

	var gap float32 = 5.0
	var timingWidth float32 = 1300
	var timingHeight float32 = 430
	const weatherWidth float32 = 170
	var trackMapWidth float32 = 900
	var rcmWidth float32 = float32(width) - (timingWidth + gap)

	// For none race session we don't display some panels
	if d.dataSrc.Session() == Messages.QualifyingSession {
		timingWidth = float32(width - 750)
		trackMapWidth = (float32(width) - gap) / 2.0
		rcmWidth = trackMapWidth
	} else if d.dataSrc.Session() != Messages.RaceSession && d.dataSrc.Session() != Messages.SprintSession {
		timingWidth = float32(width)
		trackMapWidth = (float32(width) - gap) / 2.0
		rcmWidth = trackMapWidth
	}

	// CONTROLS

	w := giu.Window(panel.Info.String()).
		Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove).
		Pos(0, 0)
	w.Layout(d.panels[panel.Info].Draw(0, 0)...)

	infoWidth, panelHeight := w.CurrentSize()

	w = giu.Window(panel.TeamRadio.String()).
		Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove).
		Pos(infoWidth+gap, 0)
	w.Layout(d.panels[panel.TeamRadio].Draw(0, 0)...)

	row1StartY := panelHeight + gap

	row2StartY := row1StartY + timingHeight + gap
	row2Height := float32(height) - row2StartY

	// ROW 1
	w = giu.Window(panel.Timing.String()).
		Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove).
		Pos(0, row1StartY).
		Size(timingWidth, timingHeight)
	w.Layout(d.panels[panel.Timing].Draw(0, 0)...)

	if d.dataSrc.Session() == Messages.RaceSession || d.dataSrc.Session() == Messages.SprintSession {
		w = giu.Window(panel.RaceControlMessages.String()).
			Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove|giu.WindowFlagsAlwaysHorizontalScrollbar|giu.WindowFlagsAlwaysVerticalScrollbar).
			Pos(timingWidth+gap, row1StartY).
			Size(rcmWidth, timingHeight)
		w.Layout(d.panels[panel.RaceControlMessages].Draw(0, 0)...)
	} else if d.dataSrc.Session() == Messages.QualifyingSession {
		w = giu.Window(panel.QualifyingImproving.String()).
			Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove).
			Pos(timingWidth+gap, row1StartY).
			Size(float32(width)-(timingWidth+gap), timingHeight)
		w.Layout(d.panels[panel.QualifyingImproving].Draw(0, 0)...)
	}

	telemetryWidth := float32(width) - gap - weatherWidth

	w = giu.Window(panel.TrackMap.String()).
		Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove).
		Pos(0, row2StartY).
		Size(trackMapWidth, row2Height)
	w.Layout(d.panels[panel.TrackMap].Draw(int(trackMapWidth), int(float32(height)-row2StartY))...)

	// For none race session don't show the catch panel but move the race control messages into it's place
	if d.dataSrc.Session() == Messages.RaceSession || d.dataSrc.Session() == Messages.SprintSession {
		w = giu.Window(panel.Catching.String()).
			Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove|giu.WindowFlagsAlwaysVerticalScrollbar).
			Pos(trackMapWidth+gap, row2StartY).
			Size(telemetryWidth, row2Height)
		w.Layout(d.panels[panel.Catching].Draw(int(telemetryWidth), int(row2Height))...)
	} else {
		w = giu.Window(panel.RaceControlMessages.String()).
			Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove|giu.WindowFlagsAlwaysHorizontalScrollbar|giu.WindowFlagsAlwaysVerticalScrollbar).
			Pos(trackMapWidth+gap, row2StartY).
			Size(telemetryWidth, row2Height)
		w.Layout(d.panels[panel.RaceControlMessages].Draw(int(telemetryWidth), int(row2Height))...)
	}
}

func (d *dataView) processData() {

	for {
		select {
		case <-d.ctx.Done():
			return

		case msg := <-d.dataSrc.Drivers():
			for x := range d.panels {
				d.panels[x].ProcessDrivers(msg)
			}

		case msg := <-d.dataSrc.Timing():
			// TODO - sometimes get empty records on shutdown so filter these out
			if msg.Position == 0 {
				continue
			}

			for x := range d.panels {
				d.panels[x].ProcessTiming(msg)
			}

		case msg := <-d.dataSrc.Event():
			d.eventLock.Lock()
			d.event = msg
			d.eventLock.Unlock()

			for x := range d.panels {
				d.panels[x].ProcessEvent(msg)
			}

		case msg := <-d.dataSrc.Time():
			for x := range d.panels {
				d.panels[x].ProcessEventTime(msg)
			}

		case msg := <-d.dataSrc.RaceControlMessages():
			for x := range d.panels {
				d.panels[x].ProcessRaceControlMessages(msg)
			}

		case msg := <-d.dataSrc.Weather():
			for x := range d.panels {
				d.panels[x].ProcessWeather(msg)
			}

		case msg := <-d.dataSrc.Radio():
			for x := range d.panels {
				d.panels[x].ProcessRadio(msg)
			}

		case msg := <-d.dataSrc.Location():
			for x := range d.panels {
				d.panels[x].ProcessLocation(msg)
			}

		case msg := <-d.dataSrc.Telemetry():
			for x := range d.panels {
				d.panels[x].ProcessTelemetry(msg)
			}
		}

		// Data has changed so force a UI redraw
		giu.Update()
	}
}
