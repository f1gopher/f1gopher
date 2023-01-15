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
	"bytes"
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto/v2"
	"sync"
	"sync/atomic"
	"time"
)

type teamRadio struct {
	audioPlayer *oto.Context
	exitSession atomic.Bool
	wg          sync.WaitGroup

	radioMsgs     []Messages.Radio
	radioMsgsLock sync.Mutex
	radioName     string

	isMuted bool
}

const noRadioMessage = "<no one>"

func CreateTeamRadio() Panel {
	return &teamRadio{}
}

func (t *teamRadio) ProcessTiming(data Messages.Timing)                          {}
func (t *teamRadio) ProcessEventTime(data Messages.EventTime)                    {}
func (t *teamRadio) ProcessEvent(data Messages.Event)                            {}
func (t *teamRadio) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (t *teamRadio) ProcessWeather(data Messages.Weather)                        {}
func (t *teamRadio) ProcessLocation(data Messages.Location)                      {}

func (t *teamRadio) Type() Type { return TeamRadio }

func (t *teamRadio) Init(dataSrc f1gopherlib.F1GopherLib) {
	// Clear previous session data
	t.radioName = noRadioMessage
	t.radioMsgs = make([]Messages.Radio, 0)
	t.exitSession.Store(false)

	// Create a new audio player each time so we don't unpause and continue playing audio from the last session
	var err error
	var ready chan struct{}
	t.audioPlayer, ready, err = oto.NewContext(48000, 2, 2)
	if err != nil {
		t.audioPlayer = nil
		// TODO - log error
	}
	<-ready

	go t.playTeamRadio()
}

func (t *teamRadio) ProcessRadio(data Messages.Radio) {
	t.radioMsgsLock.Lock()
	t.radioMsgs = append(t.radioMsgs, data)
	t.radioMsgsLock.Unlock()
}

func (t *teamRadio) Close() {
	// Tell audio player to pause and then wait for it to finish
	t.exitSession.Store(true)
	t.audioPlayer.Suspend()
	t.wg.Wait()
	t.audioPlayer = nil
}

func (t *teamRadio) Draw(width int, height int) (widgets []giu.Widget) {
	return []giu.Widget{
		giu.Row(
			giu.Checkbox("Mute Radio", &t.isMuted),
			giu.Labelf("Playing: %s", t.radioName),
		),
	}
}

func (t *teamRadio) playTeamRadio() {
	t.wg.Add(1)

	// If there was an error creating the audio player then do nothing
	if t.audioPlayer == nil {
		return
	}

	for !t.exitSession.Load() {

		if len(t.radioMsgs) > 0 {
			t.radioMsgsLock.Lock()
			currentMsg := t.radioMsgs[0]
			t.radioMsgs = t.radioMsgs[1:]
			t.radioMsgsLock.Unlock()

			// If we aren't muted then play the current message
			if !t.isMuted && t.play(currentMsg) {
				return
			}
		}

		time.Sleep(time.Second * 1)
	}

	t.wg.Done()
}

func (t *teamRadio) play(currentMsg Messages.Radio) bool {
	// Handle any dodgy mp3 data that has been corrupted by just ignoring the error and not falling over
	defer func() {
		if r := recover(); r != nil {
		}
	}()

	d, err := mp3.NewDecoder(bytes.NewReader(currentMsg.Msg))
	if err != nil {
		return true
	}

	t.radioName = currentMsg.Driver

	p := t.audioPlayer.NewPlayer(d)
	defer p.Close()
	p.Play()

	for {
		time.Sleep(time.Millisecond * 500)
		if !p.IsPlaying() {
			break
		}
	}

	// Clear the display name now the message has finished
	t.radioName = noRadioMessage
	return false
}
