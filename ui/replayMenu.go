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
	"fmt"
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
)

type replayMenu struct {
	changeView func(newView screen, info any)
	history    []any
}

func (r *replayMenu) draw(width int, height int) {
	menuWidth := float32(600.0)
	menuHeight := float32(500.0)
	posX := (float32(width) - menuWidth) / 2
	posY := (float32(height) - menuHeight) / 2
	buttonWidth := menuWidth - 40
	buttonHeight := float32(20)

	giu.Window("Replay Menu").
		Pos(posX, posY).
		Size(menuWidth, menuHeight).
		Flags(giu.WindowFlagsNoResize|giu.WindowFlagsNoMove|giu.WindowFlagsNoCollapse).
		RegisterKeyboardShortcuts(
			giu.WindowShortcut{Key: giu.KeyEscape, Callback: func() { r.changeView(MainMenu, nil) }},
		).
		Layout(
			giu.Child().Size(menuWidth, menuHeight-84).Layout(
				giu.RangeBuilder("Buttons", r.history, func(i int, v interface{}) giu.Widget {
					session := v.(f1gopherlib.RaceEvent)
					// Pre-season test session don't have a useful url so we can't replay them
					return giu.
						Button(fmt.Sprintf("%s %s - %s", session.EventTime.Format("2006"), session.Name, session.Type.String())).
						Size(buttonWidth, buttonHeight).
						OnClick(func() {
							r.changeView(Replay, &session)
						}).
						Disabled(session.Type == Messages.PreSeasonSession)
				})),
			giu.Dummy(1, 20),
			giu.Button("Back").OnClick(func() {
				r.changeView(MainMenu, nil)
			}),
		)
}
