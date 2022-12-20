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
	"f1gopher/ui/panel"
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"sync"
)

type dataView struct {
	dataSrc f1gopherlib.F1GopherLib

	changeView func(newView screen, info any)

	panels []panel.Panel

	event     Messages.Event
	eventLock sync.Mutex
}

func (d *dataView) init(dataSrc f1gopherlib.F1GopherLib) {
	d.dataSrc = dataSrc

	for x := range d.panels {
		d.panels[x].Init(dataSrc)
	}

	// Listen for and handle data messages in the background
	go d.processData()
}

func (d *dataView) draw(width int, height int) {
	var xPos float32 = 0.0
	var yPos float32 = 0.0

	for x := range d.panels {
		title, widgets := d.panels[x].Draw()
		w := giu.Window(title).
			Flags(giu.WindowFlagsNoDecoration|giu.WindowFlagsNoMove).
			Pos(xPos, yPos)
		w.Layout(widgets...)

		// Make the panels stack vertically with no overlap
		_, panelHeight := w.CurrentSize()
		yPos += panelHeight
	}
}

func (d *dataView) processData() {

	for {
		select {
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
		}

		// Data has changed so force a UI redraw
		giu.Update()
	}
}
