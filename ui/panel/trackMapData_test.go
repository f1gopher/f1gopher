package panel

import (
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/f1gopher/f1gopherlib/flowControl"
	"github.com/f1gopher/f1gopherlib/parser"
	"testing"
	"time"
)

type blah struct {
	t *testing.T
}

func (b blah) Write(p []byte) (n int, err error) {
	b.t.Logf("%s", string(p))
	return len(p), nil
}

func TestCreateTrackMaps(t *testing.T) {
	mapStore := CreateTrackMapStore()
	mapStore.tracks = map[string]*trackInfo{}
	//mapStore.targetDriver = 44

	//f1gopherlib.SetLogOutput(blah{t: t})

	history := f1gopherlib.RaceHistory()
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	for _, session := range history {

		if session.Type != Messages.Practice1Session {
			continue
		}

		data, err := f1gopherlib.CreateReplay(
			parser.Location|parser.Timing,
			session,
			"../../.cache",
			flowControl.StraightThrough)

		if err != nil {
			continue
		}

		mapStore.SelectTrack(data.Track(), session.EventTime.Year())

		exists, _, _, _ := mapStore.MapAvailable(100, 100)
		if exists {
			continue
		}

		ticker := time.NewTicker(360 * time.Second)

		t.Logf("%v - Processing track: %s %s...", time.Now().Format("15:04:05"), data.Track(), data.Session().String())

		exit := false
		for !exit {
			select {
			case <-ticker.C:
				t.Logf("%v - Timeout for track: %s", time.Now().Format("15:04:05"), data.Track())
				exit = true

			case msg := <-data.Location():
				mapStore.ProcessLocation(msg)

			case msg := <-data.Timing():
				mapStore.ProcessTiming(msg)
			}

			if mapStore.trackReady && mapStore.pitlaneReady {
				ticker.Stop()
				t.Logf("%v - Finished track: %s", time.Now().Format("15:04:05"), data.Track())
				break
			}
		}

		mapStore.writeToFile("./trackMapData2.go")
	}

	t.Log("Done")
}
