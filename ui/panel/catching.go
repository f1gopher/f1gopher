package panel

import (
	"fmt"
	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"golang.org/x/image/colornames"
	"image/color"
	"sort"
	"time"
)

type catchingInfo struct {
	color       color.RGBA
	name        string
	position    int
	visible     bool
	lapTimes    []time.Duration
	gapToLeader time.Duration
	tire        Messages.TireType
	lapsOnTire  int
}

type catching struct {
	driverData map[int]*catchingInfo
	lap        int

	driverNames []string

	selectedDriver1       int32
	selectedDriver1Number int
	selectedDriver2       int32
	selectedDriver2Number int

	selectedDriver3       int32
	selectedDriver3Number int
	selectedDriver4       int32
	selectedDriver4Number int

	table1 *giu.TableWidget
	table2 *giu.TableWidget

	config PanelConfig
}

func CreateCatching() Panel {
	return &catching{}
}

func (c *catching) ProcessEventTime(data Messages.EventTime)                    {}
func (c *catching) ProcessRaceControlMessages(data Messages.RaceControlMessage) {}
func (c *catching) ProcessWeather(data Messages.Weather)                        {}
func (c *catching) ProcessRadio(data Messages.Radio)                            {}
func (c *catching) ProcessLocation(data Messages.Location)                      {}
func (c *catching) ProcessTelemetry(data Messages.Telemetry)                    {}

func (c *catching) Type() Type { return Catching }

func (c *catching) Init(dataSrc f1gopherlib.F1GopherLib, config PanelConfig) {
	c.driverData = map[int]*catchingInfo{}
	c.lap = 0
	c.config = config
	c.driverNames = []string{}

	c.selectedDriver1 = NothingSelected
	c.selectedDriver1Number = NothingSelected
	c.selectedDriver2 = NothingSelected
	c.selectedDriver2Number = NothingSelected

	c.selectedDriver3 = NothingSelected
	c.selectedDriver3Number = NothingSelected
	c.selectedDriver4 = NothingSelected
	c.selectedDriver4Number = NothingSelected

	c.table1 = giu.Table().FastMode(true).Flags(giu.TableFlagsResizable | giu.TableFlagsSizingFixedSame)
	c.table2 = giu.Table().FastMode(true).Flags(giu.TableFlagsResizable | giu.TableFlagsSizingFixedSame)
}

func (c *catching) Close() {}

func (c *catching) ProcessDrivers(data Messages.Drivers) {
	for x := range data.Drivers {
		driver := &catchingInfo{
			color:    data.Drivers[x].Color,
			name:     data.Drivers[x].ShortName,
			lapTimes: []time.Duration{},
			visible:  true,
		}
		c.driverData[data.Drivers[x].Number] = driver

		c.driverNames = append(c.driverNames, data.Drivers[x].ShortName)
	}

	sort.Strings(c.driverNames)
}

func (c *catching) ProcessEvent(data Messages.Event) {
	c.lap = data.CurrentLap
}

func (c *catching) ProcessTiming(data Messages.Timing) {
	driverInfo, exists := c.driverData[data.Number]
	if !exists {
		return
	}

	driverInfo.position = data.Position
	driverInfo.gapToLeader = data.GapToLeader
	driverInfo.tire = data.Tire
	driverInfo.lapsOnTire = data.LapsOnTire

	// TODO - when the safety car comes out we don't get a lap time - brazil 2022
	// TODO - we don't get a lap time for the first lap - try calculate one in the lib?
	if data.LastLap < 2 {
		return
	}

	// We don't get a lap time for the first lap
	if len(driverInfo.lapTimes) == 0 || data.Lap == len(driverInfo.lapTimes)+1 {
		// Pad with the lap we never got
		if data.Lap == 2 {
			driverInfo.lapTimes = append(driverInfo.lapTimes, 0)
		}

		driverInfo.lapTimes = append(driverInfo.lapTimes, data.LastLap)
	}
}

func (c *catching) Draw(width int, height int) (widgets []giu.Widget) {

	if c.selectedDriver1 != NothingSelected && c.selectedDriver2 != NothingSelected {
		topRow1, rows := c.driverComparison2(c.selectedDriver1Number, c.selectedDriver2Number)

		c.table1.Columns(topRow1...)
		c.table1.Rows(rows...)
	}
	if c.selectedDriver3 != NothingSelected && c.selectedDriver4 != NothingSelected {
		topRow1, rows := c.driverComparison2(c.selectedDriver3Number, c.selectedDriver4Number)

		c.table2.Columns(topRow1...)
		c.table2.Rows(rows...)
	}

	driverName1 := "<none>"
	if c.selectedDriver1 != NothingSelected {
		driverName1 = c.driverNames[c.selectedDriver1]
	}
	driverName2 := "<none>"
	if c.selectedDriver2 != NothingSelected {
		driverName2 = c.driverNames[c.selectedDriver2]
	}
	driverName3 := "<none>"
	if c.selectedDriver3 != NothingSelected {
		driverName3 = c.driverNames[c.selectedDriver3]
	}
	driverName4 := "<none>"
	if c.selectedDriver4 != NothingSelected {
		driverName4 = c.driverNames[c.selectedDriver4]
	}

	return []giu.Widget{
		giu.Row(
			giu.ArrowButton(giu.DirectionLeft).OnClick(func() {
				c.config.SetPredictedPitstopTime(c.config.PredictedPitstopTime() - (time.Millisecond * 100))
			}),
			giu.Labelf("Pitstop Time: %5s", c.config.PredictedPitstopTime()),
			giu.ArrowButton(giu.DirectionRight).OnClick(func() {
				c.config.SetPredictedPitstopTime(c.config.PredictedPitstopTime() + (time.Millisecond * 100))
			})),
		giu.Dummy(0, 20),
		giu.Row(
			giu.Combo("Driver 1", driverName1, c.driverNames, &c.selectedDriver1).OnChange(func() {
				for num, driver := range c.driverData {
					if driver.name == c.driverNames[c.selectedDriver1] {
						c.selectedDriver1Number = num
						break
					}
				}
			}).Size(100),
			giu.Combo("Driver 2", driverName2, c.driverNames, &c.selectedDriver2).OnChange(func() {
				for num, driver := range c.driverData {
					if driver.name == c.driverNames[c.selectedDriver2] {
						c.selectedDriver2Number = num
						break
					}
				}
			}).Size(100),
		),
		c.table1,
		giu.Dummy(0, 20),
		giu.Row(
			giu.Combo("Driver 3", driverName3, c.driverNames, &c.selectedDriver3).OnChange(func() {
				for num, driver := range c.driverData {
					if driver.name == c.driverNames[c.selectedDriver3] {
						c.selectedDriver3Number = num
						break
					}
				}
			}).Size(100),
			giu.Combo("Driver 4", driverName4, c.driverNames, &c.selectedDriver4).OnChange(func() {
				for num, driver := range c.driverData {
					if driver.name == c.driverNames[c.selectedDriver4] {
						c.selectedDriver4Number = num
						break
					}
				}
			}).Size(100),
		),
		c.table2,
	}
}

func (c *catching) driverComparison2(driver1Number int, driver2Number int) ([]*giu.TableColumnWidget, []*giu.TableRowWidget) {
	driver1 := c.driverData[driver1Number]
	driver2 := c.driverData[driver2Number]

	first := driver1
	second := driver2
	if first.position > second.position {
		first = driver2
		second = driver1
	}

	topRow := []*giu.TableColumnWidget{
		giu.TableColumn("Driver").InnerWidthOrWeight(41),
		giu.TableColumn("Pos").InnerWidthOrWeight(41),
	}

	driver1Row := []giu.Widget{}
	driver1Row = append(driver1Row, giu.Style().SetColor(giu.StyleColorText, first.color).To(
		giu.Labelf("%s", first.name)))
	driver1Row = append(driver1Row, giu.Labelf("%d", first.position))

	driver2Row := []giu.Widget{}
	driver2Row = append(driver2Row, giu.Style().SetColor(giu.StyleColorText, second.color).To(
		giu.Labelf("%s", second.name)))
	driver2Row = append(driver2Row, giu.Labelf("%d", second.position))

	for x := c.lap - 5; x < c.lap; x++ {
		if x < 1 {
			continue
		}

		topRow = append(topRow, giu.TableColumn(fmt.Sprintf("%d", x)).InnerWidthOrWeight(timeWidth))
		if len(first.lapTimes) < x {
			driver1Row = append(driver1Row, giu.Label("-"))
		} else {
			driver1Row = append(driver1Row, giu.Labelf("%s", fmtDuration(first.lapTimes[x-1])))
		}

		if len(first.lapTimes) < x || len(second.lapTimes) < x {
			driver2Row = append(driver2Row, giu.Label("-"))
		} else {
			gap := fmtDuration(second.lapTimes[x-1] - first.lapTimes[x-1])
			color := colornames.Green
			if second.lapTimes[x-1]-first.lapTimes[x-1] > 0 {
				color = colornames.Red
			}

			driver2Row = append(driver2Row, giu.Style().SetColor(giu.StyleColorText, color).To(
				giu.Labelf("%s", gap)))
		}
	}

	gap := second.gapToLeader - first.gapToLeader

	topRow = append(topRow, giu.TableColumn("Gap").InnerWidthOrWeight(timeWidth))
	if gap >= c.config.PredictedPitstopTime() {
		driver1Row = append(driver1Row, giu.Style().SetColor(giu.StyleColorText, colornames.Green).To(giu.Label("  Can Pit")))
	} else {
		driver1Row = append(driver1Row, giu.Label("-"))
	}
	driver2Row = append(driver2Row, giu.Labelf("%s", fmtDuration(gap)))

	topRow = append(topRow, giu.TableColumn("Tire").InnerWidthOrWeight(timeWidth))
	driver1Row = append(driver1Row, giu.Style().SetColor(giu.StyleColorText, tireColor(first.tire)).To(giu.Label(first.tire.String())))
	driver2Row = append(driver2Row, giu.Style().SetColor(giu.StyleColorText, tireColor(second.tire)).To(giu.Label(second.tire.String())))

	topRow = append(topRow, giu.TableColumn("Laps On Tire").InnerWidthOrWeight(100))
	driver1Row = append(driver1Row, giu.Labelf("%d", first.lapsOnTire))
	driver2Row = append(driver2Row, giu.Labelf("%d", second.lapsOnTire))

	var rows []*giu.TableRowWidget
	rows = append(rows, giu.TableRow(driver1Row...))
	rows = append(rows, giu.TableRow(driver2Row...))
	return topRow, rows
}
