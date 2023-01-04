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
	mapStore.tracks = map[string][]*trackInfo{}
	//mapStore.targetDriver = 44

	//f1gopherlib.SetLogOutput(blah{t: t})

	history := f1gopherlib.RaceHistory()
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	for _, session := range history {

		// Sessions before 2020 don't have SessionData files so we have no segment info to work out car locations
		if session.EventTime.Year() < 2020 {
			continue
		}
		// We only need one session type
		if session.Type != Messages.Practice1Session &&
			session.Type != Messages.Practice2Session &&
			session.Type != Messages.Practice3Session {
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

		mapStore.SelectTrack(data.Track(), session.TrackYearCreated)

		exists, _, _, _ := mapStore.MapAvailable(100, 100)
		if exists {
			data.Close()
			continue
		}

		ticker := time.NewTicker(30 * time.Second)

		t.Logf("Processing track: using data for %d for session %d %s %s...", session.EventTime.Year(), session.TrackYearCreated, data.Track(), data.Session().String())

		exit := false
		for !exit {
			select {
			case <-ticker.C:
				t.Log("\tTimeout for track")
				exit = true

			case msg := <-data.Location():
				mapStore.ProcessLocation(msg)

			case msg := <-data.Timing():
				mapStore.ProcessTiming(msg)
			}

			if mapStore.trackReady && mapStore.pitlaneReady {
				ticker.Stop()
				t.Log("\tFinished track")
				break
			}
		}

		data.Close()
	}

	mapStore.writeToFile("./trackMapData2.go")

	t.Log("Done")
}
