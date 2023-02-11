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

package panel

import (
	"github.com/AllenDang/imgui-go"
)

type channelDisplaySelectWidget struct {
	id       string
	channels *[]channelConfig
	plot     *plot
}

func (c *channelDisplaySelectWidget) Build() {
	imgui.PushItemWidth(100)
	if imgui.BeginCombo("Channels", "Select") {
		for x := range *c.channels {
			if imgui.Checkbox((*c.channels)[x].name, &(*c.channels)[x].enabled) {
				c.plot.refreshBackground()
			}
		}
		imgui.EndCombo()
	}
	imgui.PopItemWidth()
}
