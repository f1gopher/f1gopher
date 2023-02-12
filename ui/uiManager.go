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
	"f1gopher/ui/webTimingView"
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/flowControl"
	"github.com/f1gopher/f1gopherlib/parser"
	"go.uber.org/zap"
	"sync"
	"time"
)

type screen int

const (
	MainMenu screen = iota
	ReplayMenu
	Live
	Replay
	DebugReplay
	OptionsMenu
	Quit
)

type drawableScreen interface {
	draw(width int, height int)
}

type dataScreen interface {
	drawableScreen

	init(dataSrc f1gopherlib.F1GopherLib)
	close()
}

type Manager struct {
	logger          *zap.SugaredLogger
	wnd             *giu.MasterWindow
	view            screen
	previousView    screen
	currentSession  *f1gopherlib.RaceEvent
	debugReplayFile string
	config          config

	mainMenu    drawableScreen
	replayMenu  drawableScreen
	optionsMenu drawableScreen
	live        dataScreen
	replay      dataScreen
	debugReplay dataScreen

	webTiming *webTimingView.WebTiming

	shutdownWg  sync.WaitGroup
	ctxShutdown context.CancelFunc
	ctx         context.Context
}

const dataSources = parser.EventTime | parser.Timing | parser.Event | parser.RaceControl |
	parser.TeamRadio | parser.Weather | parser.Location | parser.Telemetry | parser.Drivers

func Create(logger *zap.SugaredLogger, wnd *giu.MasterWindow, config config, autoLive bool) *Manager {
	manager := Manager{
		logger:       logger,
		wnd:          wnd,
		view:         MainMenu,
		previousView: MainMenu,
		config:       config,
	}

	// Context to shutdown go routines
	manager.ctx, manager.ctxShutdown = context.WithCancel(context.Background())

	main := mainMenu{
		changeView: manager.changeView,
		config:     &manager.config,
		shutdownWg: &manager.shutdownWg,
		ctx:        manager.ctx,
	}
	// Refresh the current and next session regularly for the main menu
	main.updateSessionState()
	manager.mainMenu = &main
	r := replayMenu{
		changeView: manager.changeView,
	}
	for _, x := range f1gopherlib.RaceHistory() {
		r.history = append(r.history, x)
	}
	manager.replayMenu = &r
	manager.optionsMenu = &optionsMenu{
		changeView: manager.changeView,
		config:     &manager.config,
	}

	manager.webTiming = webTimingView.CreateWebTimingView(manager.shutdownWg, manager.ctx, config.webTimingAddresses)
	if manager.config.webTimingViewEnabled {
		manager.webTiming.Start()
	}

	manager.live = createDataView(manager.webTiming, manager.changeView)
	manager.replay = createDataView(manager.webTiming, manager.changeView)
	manager.debugReplay = &debugReplayView{dataView{changeView: manager.changeView}}

	// Redraw the main menu screen every second to update the countdown and current session UI
	go manager.mainMenuRefresh()

	// If the application is closed using the window/os then shutdown all go routines properly
	wnd.SetCloseCallback(func() bool {
		manager.shutdown()
		return true
	})

	// If there is a live session currently in progress then display it
	if autoLive && main.liveSession != nil {
		manager.view = Live
	}

	return &manager
}

func (u *Manager) Loop() {
	width, height := u.wnd.GetSize()

	switch u.view {
	case MainMenu:
		u.mainMenu.draw(width, height)

	case ReplayMenu:
		u.replayMenu.draw(width, height)

	case OptionsMenu:
		u.optionsMenu.draw(width, height)

	case Replay:
		u.replay.draw(width, height)

	case Live:
		u.live.draw(width, height)

	case DebugReplay:
		u.debugReplay.draw(width, height)

	case Quit:
		u.shutdown()
		u.wnd.SetShouldClose(true)

	default:
		panic("Unhandled view")
	}
}

func (u *Manager) changeView(newView screen, info any) {

	// If we are stopping a dataview then clear the web timing display
	if u.view == Live && newView != Live {
		u.live.close()
		u.webTiming.Pause()
	}

	if u.view == Replay && newView != Replay {
		u.replay.close()
		u.webTiming.Pause()
	}

	if u.view == DebugReplay && newView != DebugReplay {
		u.debugReplay.close()
		u.webTiming.Pause()
	}

	// If we have edited the config then check if we need to enable/disable the web display
	if u.view == OptionsMenu && newView != OptionsMenu {
		if u.config.webTimingViewEnabled {
			u.webTiming.Start()
		} else {
			u.webTiming.Stop()
		}
	}

	switch newView {
	case Live:
		u.currentSession = info.(*f1gopherlib.RaceEvent)
		data, err := f1gopherlib.CreateLive(dataSources, "", u.config.sessionCache())
		if err != nil {
			u.logger.Errorln("Starting live session", err)
			return
		}
		u.live.init(data)

	case Replay:
		u.currentSession = info.(*f1gopherlib.RaceEvent)
		data, err := f1gopherlib.CreateReplay(
			dataSources,
			*u.currentSession,
			u.config.sessionCache(),
			flowControl.Realtime)

		if err != nil {
			u.logger.Errorln("Starting replay session", err)
			return
		}
		u.replay.init(data)

	case DebugReplay:
		u.debugReplayFile = info.(string)
		data, err := f1gopherlib.CreateDebugReplay(dataSources, u.debugReplayFile, flowControl.Realtime)
		if err != nil {
			u.logger.Errorln("Starting debug replay session", err)
			return
		}
		u.debugReplay.init(data)
	}

	u.previousView = u.view
	u.view = newView

	giu.Update()
}

func (u *Manager) mainMenuRefresh() {
	u.shutdownWg.Add(1)
	defer u.shutdownWg.Done()

	// Trigger a refresh every second when the main menu is displayed so that the session state display updates
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-u.ctx.Done():
				return
			case <-ticker.C:
				if u.view == MainMenu {
					giu.Update()
				}
			}
		}
	}()
}

func (u *Manager) shutdown() {
	u.logger.Infoln("Shutting down...")

	// Tell the currently displayed (if any) view/panels to close
	if u.view == Live {
		u.live.close()
	}
	if u.view == Replay {
		u.replay.close()
	}
	if u.view == DebugReplay {
		u.debugReplay.close()
	}

	// Tell all go routines to shutdown and wait for them to complete
	u.ctxShutdown()
	u.shutdownWg.Wait()

	u.logger.Infoln("Shutdown complete.")
}
