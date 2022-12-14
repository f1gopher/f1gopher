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
	"fmt"
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/sqweek/dialog"
	"sync"
	"time"
)

type mainMenu struct {
	changeView  func(newView screen, info any)
	config      *config
	liveSession *f1gopherlib.RaceEvent
	nextSession *f1gopherlib.RaceEvent
	sessionLock sync.Mutex
	shutdownWg  *sync.WaitGroup
	ctx         context.Context
}

func (m *mainMenu) updateSessionState() {
	// get the initial state so we know what is happening before we display anything
	m.getSessionState()

	go func() {
		m.shutdownWg.Add(1)
		defer m.shutdownWg.Done()

		ticker := time.NewTicker(1 * time.Minute)
		go func() {
			for {
				select {
				case <-m.ctx.Done():
					return
				case <-ticker.C:
					m.getSessionState()
				}
			}
		}()
	}()
}

func (m *mainMenu) getSessionState() {
	liveSession, nextSession, hasLiveSession, hasNextSession := f1gopherlib.HappeningSessions()

	m.sessionLock.Lock()
	if !hasLiveSession {
		m.liveSession = nil
	} else {
		m.liveSession = &liveSession
	}

	if !hasNextSession {
		m.nextSession = nil
	} else {
		m.nextSession = &nextSession
	}
	m.sessionLock.Unlock()
}

func (m *mainMenu) draw(width int, height int) {
	menuWidth := float32(200.0)
	menuHeight := float32(150.0)
	posX := (float32(width) - menuWidth) / 2
	posY := (float32(height) - menuHeight) / 2
	buttonWidth := menuWidth - 15
	buttonHeight := float32(20)

	var debugReplayBtn giu.Widget
	if m.config.showDebugReplay {
		debugReplayBtn = giu.Button("Debug Replay").Size(buttonWidth, buttonHeight).OnClick(func() {
			file, err := dialog.File().
				Title("Select a Debug Replay file").
				Filter("Debug Replay", "*.txt").
				SetStartDir(".").
				Load()

			if err != nil {
				return
			}
			m.changeView(DebugReplay, file)
		})
	}

	var sessionName string
	var sessionStart time.Time

	// Get the current/next session info
	m.sessionLock.Lock()
	isLive := m.liveSession != nil
	session := m.liveSession

	if !isLive {
		session = m.nextSession
	}

	if session != nil {
		if session.Type != Messages.PreSeasonSession {
			sessionName = fmt.Sprintf("%s - %s", session.Name, session.Type.String())
		} else {
			sessionName = fmt.Sprintf("%s - %s", session.TrackName, session.Type.String())
		}
		sessionStart = session.EventTime
	}
	m.sessionLock.Unlock()

	// Menu UI
	giu.Window("Main Menu").
		Pos(posX, posY).
		Flags(giu.WindowFlagsNoResize|giu.WindowFlagsNoMove|giu.WindowFlagsNoCollapse|giu.WindowFlagsAlwaysAutoResize).
		Layout(
			giu.PrepareMsgbox(),
			giu.Button("Live").Size(buttonWidth, buttonHeight).OnClick(func() {
				m.changeView(Live, nil)
			}).Disabled(!isLive),
			giu.Button("Replay").Size(buttonWidth, buttonHeight).OnClick(func() {
				m.changeView(ReplayMenu, nil)
			}),
			debugReplayBtn,
			giu.Button("Options").Size(buttonWidth, buttonHeight).OnClick(func() {
				m.changeView(OptionsMenu, nil)
			}),
			giu.Button("Quit").Size(buttonWidth, buttonHeight).OnClick(func() {
				giu.Msgbox("Quit?", "Are you sure you want to quit?").
					Buttons(giu.MsgboxButtonsYesNo).
					ResultCallback(func(result giu.DialogResult) {
						switch result {
						case giu.DialogResultYes:
							m.changeView(Quit, nil)
						}
					})
			}),
		)

	// Current/Next session status UI
	if sessionName != "" {
		if isLive {
			giu.Window("Current Session").
				Pos(float32(width-405), float32(height-75)).
				Size(400, 70).
				Flags(giu.WindowFlagsNoResize|giu.WindowFlagsNoMove|giu.WindowFlagsNoCollapse).
				Layout(
					giu.Label(sessionName),
					giu.Label(fmt.Sprintf(
						"Session Started at: %v",
						// Session time is in UTC so convert to local time for display
						m.liveSession.EventTime.In(time.Now().Location()).Format("15:04"))),
				)
		} else {
			// Work out time until session starts in days, hours and minutes
			countdown := sessionStart.Sub(time.Now().UTC())
			minutes := int(countdown.Minutes())
			days := minutes / (60 * 24)
			minutes = minutes % (60 * 24)
			hours := minutes / 60
			minutes = minutes % 60
			label := fmt.Sprintf("Starts in: %.2d:%.2d", hours, minutes)
			if days > 0 {
				label = fmt.Sprintf("Starts in: %d days %.2d:%.2d", days, hours, minutes)
			}

			giu.Window("Next Session").
				Pos(float32(width-405), float32(height-75)).
				Size(400, 70).
				Flags(giu.WindowFlagsNoResize|giu.WindowFlagsNoMove|giu.WindowFlagsNoCollapse).
				Layout(
					giu.Label(sessionName),
					giu.Label(label),
				)
		}
	}
}
