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
)

type dataView struct {
	dataSrc f1gopherlib.F1GopherLib

	changeView func(newView screen, info any)
}

func (d *dataView) init(dataSrc f1gopherlib.F1GopherLib) {
	d.dataSrc = dataSrc
}

func (d *dataView) draw(width int, height int) {
	giu.SingleWindow().Layout(
		giu.Label(fmt.Sprintf("%s - %s", d.dataSrc.Name(), d.dataSrc.Session().String())),

		giu.Button("Back").OnClick(func() { d.changeView(MainMenu, nil) }),
	)
}
