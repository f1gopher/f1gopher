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
)

type replayView struct {
	dataView
}

func createReplayView(webView panel.Panel, changeView func(newView screen, info any)) dataScreen {
	view := replayView{
		dataView{
			changeView: changeView,
			panels: []panel.Panel{
				panel.CreateInformation(func() { changeView(MainMenu, nil) }),
				panel.CreateTiming(),
			},
		},
	}

	view.panels = append(view.panels, webView)

	return &view
}
