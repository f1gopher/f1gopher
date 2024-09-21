package panel

import (
	"fmt"
	"image/color"
	"sort"
	"time"

	"github.com/AllenDang/giu"
	"github.com/f1gopher/f1gopherlib"
	"github.com/f1gopher/f1gopherlib/Messages"
	"golang.org/x/image/colornames"
)

type catchingInfo struct {
	color       color.RGBA
	name        string
	team        string
	position    int
	visible     bool
	lapTimes    []time.Duration
	gapToLeader time.Duration
	tire        Messages.TireType
	lapsOnTire  int
}

type trackerType int

const (
	AnotherDriver trackerType = iota
	CarInfront
	CarBehind
	CarInfrontBehind
	Leader
	Teammate
)

var trackerModes = []string{"Another Driver", "Car Infront", "Car Behind", "Cars Infront & Behind", "Leader", "Teammate"}

func (t trackerType) String() string {
	return trackerModes[t]
}

type catchingBlock struct {
	mode         trackerType
	modeDropdown int32

	selectedDriver1Index  int32
	selectedDriver1Number int
	selectedDriver2Index  int32
	selectedDriver2Number int

	selectedDriver3Number int

	table *giu.TableWidget
}

type catching struct {
	driverData map[int]*catchingInfo
	lap        int

	driverNames []string
	blocks      []catchingBlock
	driverOrder []int

	removes []int

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
	c.blocks = make([]catchingBlock, 0)
	c.driverOrder = nil
	c.removes = []int{}
}

func (c *catching) Close() {}

func (c *catching) ProcessDrivers(data Messages.Drivers) {
	for x := range data.Drivers {
		driver := &catchingInfo{
			color:    data.Drivers[x].Color,
			name:     data.Drivers[x].ShortName,
			team:     data.Drivers[x].Team,
			lapTimes: []time.Duration{},
			visible:  true,
		}
		c.driverData[data.Drivers[x].Number] = driver

		c.driverNames = append(c.driverNames, data.Drivers[x].ShortName)
	}

	sort.Strings(c.driverNames)

	// +1 because 0 will be empty because positions aren't zero based
	c.driverOrder = make([]int, len(c.driverNames)+1)
	for x := range data.Drivers {
		c.driverOrder[data.Drivers[x].StartPosition] = data.Drivers[x].Number
	}
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

	c.driverOrder[data.Position] = data.Number

	// TODO - when the safety car comes out we don't get a lap time - brazil 2022
	// TODO - we don't get a lap time for the first lap - try calculate one in the lib?
	if data.LastLap < 2 {
		return
	}

	// We don't get a lap time for the first lap
	if len(driverInfo.lapTimes) == 0 || data.Lap == len(driverInfo.lapTimes)+1 {
		// Pad with the lap we never got
		for x := len(driverInfo.lapTimes); x < data.Lap-1; x++ {
			driverInfo.lapTimes = append(driverInfo.lapTimes, 0)
		}

		driverInfo.lapTimes = append(driverInfo.lapTimes, data.LastLap)
	}
}

func (c *catching) Draw(width int, height int) (widgets []giu.Widget) {

	blockWidgets := []giu.Widget{
		giu.Row(
			giu.ArrowButton(giu.DirectionLeft).OnClick(func() {
				c.config.SetPredictedPitstopTime(c.config.PredictedPitstopTime() - (time.Millisecond * 500))
			}),
			giu.Labelf("Pitstop Time: %5s", c.config.PredictedPitstopTime()),
			giu.ArrowButton(giu.DirectionRight).OnClick(func() {
				c.config.SetPredictedPitstopTime(c.config.PredictedPitstopTime() + (time.Millisecond * 500))
			}),
			giu.Button("Add Tracker").OnClick(func() {
				newBlock := catchingBlock{
					mode:                  AnotherDriver,
					modeDropdown:          0,
					selectedDriver1Index:  NothingSelected,
					selectedDriver1Number: NothingSelected,
					selectedDriver2Index:  NothingSelected,
					selectedDriver2Number: NothingSelected,
					table:                 giu.Table().FastMode(true).Flags(giu.TableFlagsResizable | giu.TableFlagsSizingFixedSame),
				}

				c.blocks = append(c.blocks, newBlock)
			})),
	}

	// Iterate the removes in reverse order so when we remove something it doesn't affect the other remove indexes
	sort.Sort(sort.Reverse(sort.IntSlice(c.removes)))
	for _, x := range c.removes {
		c.blocks = append(c.blocks[:x], c.blocks[x+1:]...)
	}
	c.removes = []int{}

	for x := range c.blocks {
		block := &c.blocks[x]
		blockIndex := x

		c.update(x)

		if block.selectedDriver1Number != NothingSelected && block.selectedDriver2Number != NothingSelected {
			topRow1, rows := c.driverComparison(
				block.selectedDriver1Number,
				block.selectedDriver2Number,
				block.selectedDriver3Number,
				block.mode)

			block.table.Columns(topRow1...)
			block.table.Rows(rows...)
		}

		driverName1 := "<none>"
		if block.selectedDriver1Index != NothingSelected {
			driverName1 = c.driverNames[block.selectedDriver1Index]
		}
		driverName2 := "<none>"
		if block.selectedDriver2Index != NothingSelected {
			driverName2 = c.driverNames[block.selectedDriver2Index]
		}

		blockWidgets = append(blockWidgets, giu.Dummy(0, 5))
		if block.mode == AnotherDriver {
			blockWidgets = append(blockWidgets,
				giu.Row(
					giu.Combo("Mode", block.mode.String(), trackerModes, &block.modeDropdown).OnChange(func() {
						block.mode = trackerType(block.modeDropdown)

						if block.mode == Teammate {
							block.selectedDriver3Number = NothingSelected
							block.selectedDriver2Number = c.findTeammate(block.selectedDriver1Number)
						}
					}).Size(200),
					giu.Combo("Driver", driverName1, c.driverNames, &block.selectedDriver1Index).OnChange(func() {
						for num, driver := range c.driverData {
							if driver.name == c.driverNames[block.selectedDriver1Index] {
								block.selectedDriver3Number = NothingSelected
								block.selectedDriver1Number = num
								break
							}
						}
					}).Size(100),
					giu.Combo("Other Driver", driverName2, c.driverNames, &block.selectedDriver2Index).OnChange(func() {
						for num, driver := range c.driverData {
							if driver.name == c.driverNames[block.selectedDriver2Index] {
								block.selectedDriver3Number = NothingSelected
								block.selectedDriver2Number = num
								break
							}
						}
					}).Size(100),
					giu.Button("Remove").OnClick(func() {
						c.removes = append(c.removes, blockIndex)
					}),
				))
		} else {
			blockWidgets = append(blockWidgets,
				giu.Row(
					giu.Combo("Mode", block.mode.String(), trackerModes, &block.modeDropdown).OnChange(func() {
						block.mode = trackerType(block.modeDropdown)

						if block.mode == Teammate {
							block.selectedDriver3Number = NothingSelected
							block.selectedDriver2Number = c.findTeammate(block.selectedDriver1Number)
						}
					}).Size(200),
					giu.Combo("Driver", driverName1, c.driverNames, &block.selectedDriver1Index).OnChange(func() {
						for num, driver := range c.driverData {
							if driver.name == c.driverNames[block.selectedDriver1Index] {
								block.selectedDriver1Number = num

								if block.mode == Teammate {
									block.selectedDriver3Number = NothingSelected
									block.selectedDriver2Number = c.findTeammate(block.selectedDriver1Number)
								} else {
									c.update(x)
								}

								break
							}
						}
					}).Size(100),
					giu.Button("Remove").OnClick(func() {
						c.removes = append(c.removes, blockIndex)
					}),
				))
		}
		blockWidgets = append(blockWidgets, block.table)
	}

	return blockWidgets
}

func (c *catching) driverComparison(driver1Number int, driver2Number int, driver3Number int, mode trackerType) ([]*giu.TableColumnWidget, []*giu.TableRowWidget) {
	driver1 := c.driverData[driver1Number]
	var driver2 *catchingInfo = nil
	var driver3 *catchingInfo = nil
	// For infront and behind the driver1 is the focussed driver and should be in the middle
	if mode == CarInfrontBehind {
		if driver2Number != NoDriver {
			driver1 = c.driverData[driver2Number]
		} else {
			driver1 = nil
		}
		driver2 = c.driverData[driver1Number]
		if driver3Number != NoDriver {
			driver3 = c.driverData[driver3Number]
		}
	} else {
		driver2 = c.driverData[driver2Number]
	}

	first := driver1
	second := driver2

	// Order the drivers for these modes. Other modes the order is fixed
	if mode == Teammate || mode == AnotherDriver {
		if first.position > second.position {
			first = driver2
			second = driver1
		}
	} else if mode == CarInfront || mode == Leader {
		first = driver2
		second = driver1
	}

	topRow := []*giu.TableColumnWidget{
		giu.TableColumn("Driver").InnerWidthOrWeight(41),
		giu.TableColumn("Pos").InnerWidthOrWeight(41),
	}

	driver1Row := []giu.Widget{}
	if first != nil {
		driver1Row = append(driver1Row, giu.Style().SetColor(giu.StyleColorText, first.color).To(
			giu.Labelf("%s", first.name)))
		driver1Row = append(driver1Row, giu.Labelf("%d", first.position))
	} else {
		driver1Row = append(driver1Row, giu.Label(""))
		driver1Row = append(driver1Row, giu.Label(""))
	}

	driver2Row := []giu.Widget{}
	if second != nil {
		driver2Row = append(driver2Row, giu.Style().SetColor(giu.StyleColorText, second.color).To(
			giu.Labelf("%s", second.name)))
		driver2Row = append(driver2Row, giu.Labelf("%d", second.position))
	} else {
		driver2Row = append(driver2Row, giu.Label(""))
		driver2Row = append(driver2Row, giu.Label(""))
	}

	driver3Row := []giu.Widget{}

	if driver3 != nil {
		driver3Row = append(driver3Row, giu.Style().SetColor(giu.StyleColorText, driver3.color).To(
			giu.Labelf("%s", driver3.name)))
		driver3Row = append(driver3Row, giu.Labelf("%d", driver3.position))
	}

	defaultColumnWidth := float32(timeWidth + 2)

	if mode == CarInfrontBehind {
		for x := c.lap - 5; x < c.lap; x++ {
			if x < 1 {
				continue
			}

			topRow = append(topRow, giu.TableColumn(fmt.Sprintf("%d", x)).InnerWidthOrWeight(defaultColumnWidth))
			if first == nil || len(second.lapTimes) < x {
				driver1Row = append(driver1Row, giu.Label("-"))
			} else {
				gap := fmtDuration(first.lapTimes[x-1] - second.lapTimes[x-1])
				color := colornames.Green
				if first.lapTimes[x-1]-second.lapTimes[x-1] > 0 {
					color = colornames.Red
				}
				driver1Row = append(driver1Row, giu.Style().SetColor(giu.StyleColorText, color).To(
					giu.Labelf("%s", gap)))
			}

			if first == nil || len(first.lapTimes) < x || len(second.lapTimes) < x {
				driver2Row = append(driver2Row, giu.Label("-"))
			} else {
				driver2Row = append(driver2Row, giu.Labelf("%s", fmtDuration(second.lapTimes[x-1])))
			}

			if driver3 != nil {
				if len(second.lapTimes) < x || len(driver3.lapTimes) < x {
					driver3Row = append(driver3Row, giu.Label("-"))
				} else {
					gap := fmtDuration(driver3.lapTimes[x-1] - second.lapTimes[x-1])
					color := colornames.Green
					if driver3.lapTimes[x-1]-second.lapTimes[x-1] > 0 {
						color = colornames.Red
					}

					driver3Row = append(driver3Row, giu.Style().SetColor(giu.StyleColorText, color).To(
						giu.Labelf("%s", gap)))
				}
			}
		}
	} else {
		for x := c.lap - 5; x < c.lap; x++ {
			if x < 1 {
				continue
			}

			topRow = append(topRow, giu.TableColumn(fmt.Sprintf("%d", x)).InnerWidthOrWeight(defaultColumnWidth))
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
	}

	// Pad columns for any laps we don't yet have
	for x := len(topRow); x < 7; x++ {
		topRow = append(topRow, giu.TableColumn("").InnerWidthOrWeight(defaultColumnWidth))
		driver1Row = append(driver1Row, giu.Label(""))
		driver2Row = append(driver2Row, giu.Label(""))
		driver3Row = append(driver3Row, giu.Label(""))
	}

	var gap time.Duration
	if first != nil && second != nil {
		gap = second.gapToLeader - first.gapToLeader
	}

	topRow = append(topRow, giu.TableColumn("Gap").InnerWidthOrWeight(defaultColumnWidth))
	if gap >= c.config.PredictedPitstopTime() {
		driver1Row = append(driver1Row, giu.Style().SetColor(giu.StyleColorText, colornames.Green).To(giu.Label("  Can Pit")))
	} else {
		driver1Row = append(driver1Row, giu.Label("-"))
	}

	if gap == 0 {
		driver2Row = append(driver2Row, giu.Label("-"))
	} else {
		driver2Row = append(driver2Row, giu.Labelf("%s", fmtDuration(gap)))
	}

	topRow = append(topRow, giu.TableColumn("Tire").InnerWidthOrWeight(50))
	firstTire := "-"
	var firstTireColor color.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if first != nil {
		firstTire = first.tire.String()
		firstTireColor = tireColor(first.tire)
	}
	driver1Row = append(driver1Row, giu.Style().SetColor(giu.StyleColorText, firstTireColor).To(giu.Label(firstTire)))
	if driver2 != nil {
		driver2Row = append(driver2Row, giu.Style().SetColor(giu.StyleColorText, tireColor(second.tire)).To(giu.Label(second.tire.String())))
	} else {
		driver2Row = append(driver2Row, giu.Label(""))
	}

	var rows []*giu.TableRowWidget
	if first != nil {
		rows = append(rows, giu.TableRow(driver1Row...))
	}
	rows = append(rows, giu.TableRow(driver2Row...))

	if driver3 != nil {
		gap = driver3.gapToLeader - second.gapToLeader
		if gap == 0 {
			driver3Row = append(driver3Row, giu.Label("-"))
		} else {
			driver3Row = append(driver3Row, giu.Labelf("%s", fmtDuration(gap)))
		}
		driver3Row = append(driver3Row, giu.Style().SetColor(giu.StyleColorText, tireColor(driver3.tire)).To(giu.Label(driver3.tire.String())))
		rows = append(rows, giu.TableRow(driver3Row...))
	}

	return topRow, rows
}

func (c *catching) findTeammate(currentDriver int) int {
	if currentDriver == NothingSelected {
		return NothingSelected
	}

	for num, driver := range c.driverData {
		if num == currentDriver {
			continue
		}

		if driver.team == c.driverData[currentDriver].team {
			return num
		}
	}

	return NothingSelected
}

func (c *catching) findLeader(currentDriver int) int {
	if currentDriver == NothingSelected {
		return NothingSelected
	}

	currentPos := c.driverData[currentDriver].position

	if currentPos == 1 {
		return NothingSelected
	}

	return c.driverOrder[1]
}

func (c *catching) findCarInfront(currentDriver int) int {
	if currentDriver == NothingSelected {
		return NothingSelected
	}

	currentPos := c.driverData[currentDriver].position

	if currentPos == 1 {
		return NothingSelected
	}

	return c.driverOrder[currentPos-1]
}

func (c *catching) findCarBehind(currentDriver int) int {
	if currentDriver == NothingSelected {
		return NothingSelected
	}

	currentPos := c.driverData[currentDriver].position

	if currentPos+1 >= len(c.driverOrder) {
		return NothingSelected
	}

	return c.driverOrder[currentPos+1]
}

func (c *catching) findCarInfrontAndBehind(currentDriver int) (front int, behind int) {
	if currentDriver == NothingSelected {
		return NothingSelected, NothingSelected
	}

	currentPos := c.driverData[currentDriver].position
	front = NoDriver
	behind = NoDriver

	if currentPos != 1 {
		front = c.driverOrder[currentPos-1]
	}

	if currentPos+1 < len(c.driverOrder) {
		behind = c.driverOrder[currentPos+1]
	}

	return front, behind
}

func (c *catching) update(blockIndex int) {
	switch c.blocks[blockIndex].mode {
	case CarInfront:
		c.blocks[blockIndex].selectedDriver3Number = NothingSelected
		c.blocks[blockIndex].selectedDriver2Number = c.findCarInfront(c.blocks[blockIndex].selectedDriver1Number)
	case CarBehind:
		c.blocks[blockIndex].selectedDriver3Number = NothingSelected
		c.blocks[blockIndex].selectedDriver2Number = c.findCarBehind(c.blocks[blockIndex].selectedDriver1Number)
	case CarInfrontBehind:
		c.blocks[blockIndex].selectedDriver2Number, c.blocks[blockIndex].selectedDriver3Number = c.findCarInfrontAndBehind(c.blocks[blockIndex].selectedDriver1Number)
	case Leader:
		c.blocks[blockIndex].selectedDriver3Number = NothingSelected
		c.blocks[blockIndex].selectedDriver2Number = c.findLeader(c.blocks[blockIndex].selectedDriver1Number)
	}
}
