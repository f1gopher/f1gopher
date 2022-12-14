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

package main

import (
	"f1gopher/ui"
	"flag"
	"fmt"
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"go.uber.org/zap"
	"os"
)

func main() {

	autoLivePtr := flag.Bool("autoLive", false, "If a live session is in progress display it on startup")
	logPtr := flag.Bool("log", false, "Enable logging")
	flag.Parse()

	config := ui.NewConfig()

	var logger *zap.Logger
	if !*logPtr {
		logger = zap.NewNop()
	} else {
		// Logging goes to stderr for both the library and app
		logger, _ = zap.NewDevelopment(zap.AddStacktrace(zap.WarnLevel))
		f1gopherlib.SetLogOutput(os.Stderr)
	}
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Infof("F1Gopher v%s", version)

	wnd := giu.NewMasterWindow(
		fmt.Sprintf("F1Gopher - v%s", version),
		1024,
		768,
		0)
	uiManager := ui.Create(sugar, wnd, config, *autoLivePtr)
	wnd.Run(uiManager.Loop)
}
