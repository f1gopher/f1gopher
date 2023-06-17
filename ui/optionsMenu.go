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
	"github.com/AllenDang/giu"
	"strings"
)

type optionsMenu struct {
	changeView func(newView screen, info any)
	config     *config
}

func (o *optionsMenu) draw(width int, height int) {
	menuWidth := float32(600.0)
	menuHeight := float32(380.0)
	posX := (float32(width) - menuWidth) / 2
	posY := (float32(height) - menuHeight) / 2

	giu.Window("Options").
		Pos(posX, posY).
		Size(menuWidth, menuHeight).
		Flags(giu.WindowFlagsNoResize|giu.WindowFlagsNoMove|giu.WindowFlagsNoCollapse|giu.WindowFlagsAlwaysAutoResize).
		RegisterKeyboardShortcuts(
			giu.WindowShortcut{Key: giu.KeyEscape, Callback: func() { o.changeView(MainMenu, nil) }},
		).
		Layout(
			giu.Checkbox("Autoplay Live Session on Startup", &o.config.autoplayLive),
			giu.InputInt(&o.config.liveDelay).Size(20).Label("Live Delay (in seconds)"),
			giu.Checkbox("Cache Replay Data", &o.config.useCache),
			giu.InputText(&o.config.cacheFolder).Label("Replay Cache Folder"),
			giu.Dummy(1, 20),
			giu.Checkbox("Web Timing View Enabled", &o.config.webTimingViewEnabled),
			giu.Label("Web Timing View Addresses:"),
			// Indent the addresses
			giu.Label("      "+strings.Join(o.config.webTimingAddresses, ", ")),
			giu.InputInt(&o.config.webTimingPort).Size(40).Label("Web Timing View Port"),
			giu.Dummy(1, 20),
			giu.Checkbox("Show Debug Replay", &o.config.showDebugReplay),
			giu.Dummy(1, 20),
			giu.Dummy(1, 20),
			giu.Button("Back").OnClick(func() {
				o.changeView(MainMenu, nil)
			}),
		)
}
