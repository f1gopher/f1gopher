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
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
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

	shutdownWg  sync.WaitGroup
	ctxShutdown context.CancelFunc
	ctx         context.Context
}

const dataSources = parser.EventTime | parser.Timing | parser.Event | parser.RaceControl | parser.TeamRadio | parser.Weather

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
	manager.live = &liveView{dataView{changeView: manager.changeView}}
	manager.replay = &replayView{dataView{changeView: manager.changeView}}
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
	switch newView {
	case Live:
		u.currentSession = info.(*f1gopherlib.RaceEvent)
		data, err := f1gopherlib.CreateLive(dataSources, "", u.config.sessionCache(u.currentSession))
		if err != nil {
			u.logger.Errorln("Starting live session", err)
			return
		}
		u.live.init(data)

	case Replay:
		u.currentSession = info.(*f1gopherlib.RaceEvent)
		data, err := f1gopherlib.CreateReplay(dataSources, *u.currentSession, u.config.sessionCache(u.currentSession))
		if err != nil {
			u.logger.Errorln("Starting replay session", err)
			return
		}
		u.replay.init(data)

	case DebugReplay:
		u.debugReplayFile = info.(string)
		data, err := f1gopherlib.CreateDebugReplay(dataSources, u.debugReplayFile)
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

	// Tell all go routines to shutdown and wait for them to complete
	u.ctxShutdown()
	u.shutdownWg.Wait()

	u.logger.Infoln("Shutdown complete.")
}
