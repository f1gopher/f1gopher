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
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"sync"
)

type dataView struct {
	ctxShutdown context.CancelFunc
	ctx         context.Context

	dataSrc f1gopherlib.F1GopherLib

	changeView func(newView screen, info any)

	panels []panel.Panel

	event     Messages.Event
	eventLock sync.Mutex
}

func (d *dataView) init(dataSrc f1gopherlib.F1GopherLib) {
	d.dataSrc = dataSrc
	d.ctx, d.ctxShutdown = context.WithCancel(context.Background())

	for x := range d.panels {
		d.panels[x].Init(dataSrc)
	}

	// Listen for and handle data messages in the background
	go d.processData()
}

func (d *dataView) close() {
	if d.ctxShutdown != nil {
		d.ctxShutdown()
	}

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
	var xPos float32 = 0.0
	var yPos float32 = 0.0

	for x := range d.panels {
		title, widgets := d.panels[x].Draw()

		// Not all panels display something (web timing view)
		if widgets == nil {
			continue
		}

		var w *giu.WindowWidget
		if title == "Track Map" {
			w = giu.Window(title).
				Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove).
				Pos(xPos, yPos).
				Size(500, 500)
		} else {
			w = giu.Window(title).
				Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove).
				Pos(xPos, yPos)
		}

		w.Layout(widgets...)

		// Make the panels stack vertically with no overlap
		_, panelHeight := w.CurrentSize()
		yPos += panelHeight
	}
}

func (d *dataView) processData() {

	for {
		select {
		case <-d.ctx.Done():
			return

		case msg := <-d.dataSrc.Timing():
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
		}

		// Data has changed so force a UI redraw
		giu.Update()
	}
}
