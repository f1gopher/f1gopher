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

type fudgeFactors struct {
	rotation float64
}

var fudge = map[string]fudgeFactors{
	"Albert Park Grand Prix Circuit":              {rotation: -0.7},
	"Autodromo Enzo e Dino Ferrari":               {rotation: 0.0},
	"Autódromo Hermanos Rodríguez":                {rotation: -0.1396263},
	"Autódromo Internacional do Algarve":          {rotation: 1.832596},
	"Autodromo Internazionale del Mugello":        {rotation: 1.22173},
	"Autódromo José Carlos Pace":                  {rotation: 1.5708},
	"Autodromo Nazionale di Monza":                {rotation: 1.48353},
	"Bahrain International Circuit":               {rotation: 1.535},
	"Bahrain International Circuit - Outer Track": {rotation: 1.535},
	"Baku City Circuit":                           {rotation: 0.4014257},
	"Circuit de Barcelona-Catalunya":              {rotation: -2.129302},
	"Circuit de Monaco":                           {rotation: 1.396263},
	"Circuit de Spa-Francorchamps":                {rotation: 1.308997},
	"Circuit Gilles Villeneuve":                   {rotation: 1.832596},
	"Circuit of the Americas":                     {rotation: 0.0},
	"Circuit Paul Ricard":                         {rotation: 0.0},
	"Circuit Park Zandvoort":                      {rotation: 0.0},
	"Hungaroring":                                 {rotation: 0.7853982}, // TODO - fix
	"Istanbul Park":                               {rotation: -2.879793},
	"Jeddah Corniche Circuit":                     {rotation: -0.8726646},
	"Losail International Circuit":                {rotation: 2.1},
	"Marina Bay Street Circuit":                   {rotation: 0.0},
	"Miami International Autodrome":               {rotation: 0.0},
	"Nürburgring":                                 {rotation: 0.3},
	"Red Bull Ring":                               {rotation: 0.0},
	"Silverstone Circuit":                         {rotation: 1.5708},
	"Sochi Autodrom":                              {rotation: 0.0},
	"Suzuka Circuit":                              {rotation: 0.0},
	"Yas Marina Circuit":                          {rotation: 1.5708},
	"Las Vegas Strip Street Circuit":              {rotation: -1.5708},
}

// When running remember to modify in f1gopherlib:
//
// 1) replay.go readEntries() ticker time -> Millisecond
// 2) f1gopherlib.go increase channel sizes to 10000
// 3) replay.go CreateReplay() dataFeed channel size -> 100000

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

		// No data
		if session.EventTime.Year() == 2022 && session.TrackName == "Autodromo Enzo e Dino Ferrari" {
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

		exists, _, _, _, _ := mapStore.MapAvailable(100, 100)
		if exists {
			data.Close()
			continue
		}

		if session.TrackName == "Marina Bay Street Circuit" && session.EventTime.Year() != 2023 {
			mapStore.targetDriver = 5
		} else if session.TrackName == "Bahrain International Circuit - Outer Track" {
			mapStore.targetDriver = 0
		} else {
			mapStore.targetDriver = 44
		}

		ticker := time.NewTicker(60 * time.Second)

		t.Logf("Processing track: using data for %d for session %d %s %s...", session.EventTime.Year(), session.TrackYearCreated, data.Track(), data.Session().String())

		// Apply fudging to rotate the track and display better
		fudgeInfo, exists := fudge[mapStore.currentTrack.name]
		if exists {
			mapStore.currentTrack.rotation = fudgeInfo.rotation
		}

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

				mapStore.MapAvailable(1000, 608)

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

			mapStore.MapAvailable(1000, 608)

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
