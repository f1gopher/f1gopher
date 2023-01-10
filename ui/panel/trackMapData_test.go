package panel

import (
	"fmt"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"github.com/f1gopher/f1gopherlib/flowControl"
	"github.com/f1gopher/f1gopherlib/parser"
	"golang.org/x/image/colornames"
	"image/png"
	"log"
	"os"
	"testing"
	"time"
)

func TestCreateTrackMaps(t *testing.T) {
	mapStore := CreateTrackMapStore()
	mapStore.tracks = map[string][]*trackInfo{}

	os.Mkdir("../../track images", 0755)

	history := f1gopherlib.RaceHistory()
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	for _, session := range history {

		// Sessions before 2020 don't have SessionData files so we have no segment info to work out car locations
		if session.EventTime.Year() < 2020 {
			continue
		}

		if session.Type != Messages.RaceSession {
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

		if session.TrackName == "Marina Bay Street Circuit" {
			mapStore.targetDriver = 5
		} else if session.TrackName != "Bahrain International Circuit - Outer Track" {
			mapStore.targetDriver = 44
		}

		ticker := time.NewTicker(30 * time.Second)

		t.Logf("Processing track: using data for %d for session %d %s %s...", session.EventTime.Year(), session.TrackYearCreated, data.Track(), data.Session().String())

		exit := false
		for !exit {
			select {
			case <-ticker.C:
				t.Logf("\tTimeout for track with driver %d", mapStore.targetDriver)
				exit = true

			case msg := <-data.Location():
				mapStore.ProcessLocation(msg)

			case msg := <-data.Timing():
				mapStore.ProcessTiming(msg)
			}

			if mapStore.trackReady && mapStore.pitlaneReady {
				ticker.Stop()
				t.Logf("\tFinished track using driver %d", mapStore.targetDriver)

				mapStore.MapAvailable(500, 500)

				f, err := os.Create(fmt.Sprintf("../../track images/%s-%d.png", session.TrackName, session.TrackYearCreated))
				if err != nil {
					panic(err)
				}
				if err = png.Encode(f, mapStore.gc.GetImage()); err != nil {
					log.Printf("failed to encode: %v", err)
				}
				f.Close()

				break
			}
		}

		data.Close()
	}

	mapStore.writeToFile("./trackMapData2.go")

	t.Log("Done")
}

func TestSaveTrackMapsToDisk(t *testing.T) {
	mapStore := CreateTrackMapStore()
	mapStore.backgroundColor = colornames.Cadetblue

	os.Mkdir("../../track images", 0755)

	for trackName, tracks := range mapStore.tracks {
		for _, track := range tracks {
			mapStore.SelectTrack(trackName, track.yearCreated)

			mapStore.MapAvailable(500, 500)

			f, err := os.Create(fmt.Sprintf("../../track images/%s-%d.png", trackName, track.yearCreated))
			if err != nil {
				panic(err)
			}
			if err = png.Encode(f, mapStore.gc.GetImage()); err != nil {
				log.Printf("failed to encode: %v", err)
			}
			f.Close()
		}
	}
}
