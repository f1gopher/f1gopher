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
	"net"
	"time"
)

type config struct {
	autoplayLive          bool
	liveDelay             int32
	useCache              bool
	cacheFolder           string
	webTimingViewEnabled  bool
	webTimingAddresses    []string
	webTimingPort         int32
	showDebugReplay       bool
	predictionPitstopTime time.Duration
}

func NewConfig() config {
	c := config{
		autoplayLive:          false,
		liveDelay:             0,
		useCache:              true,
		cacheFolder:           "./.cache",
		webTimingViewEnabled:  false,
		webTimingAddresses:    nil,
		webTimingPort:         8000,
		showDebugReplay:       false,
		predictionPitstopTime: time.Second * 10,
	}

	for _, address := range c.getLocalIP() {
		c.webTimingAddresses = append(c.webTimingAddresses, fmt.Sprintf("%s:%d", address, c.webTimingPort))
	}

	return c
}

func (c *config) sessionCache() string {
	if !c.useCache {
		return ""
	}

	return c.cacheFolder
}

func (c *config) getLocalIP() []string {
	ips := []string{"localhost"}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}

func (c *config) PredictedPitstopTime() time.Duration {
	return c.predictionPitstopTime
}

func (c *config) SetPredictedPitstopTime(value time.Duration) {
	c.predictionPitstopTime = value
}
