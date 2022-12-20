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
	"github.com/f1gopher/f1gopherlib"
	"path"
)

type config struct {
	autoplayLive    bool
	liveDelay       int32
	useCache        bool
	cacheFolder     string
	showDebugReplay bool
}

func NewConfig() config {
	return config{
		autoplayLive:    false,
		liveDelay:       0,
		useCache:        true,
		cacheFolder:     "./.cache",
		showDebugReplay: false,
	}
}

func (c *config) sessionCache(session *f1gopherlib.RaceEvent) string {
	if !c.useCache {
		return ""
	}

	return path.Join(c.cacheFolder, session.Name, session.Type.String())
}
