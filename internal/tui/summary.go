package summary

import (
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"

	"github.com/bcdxn/f1cli/internal/domain"
	"github.com/bcdxn/f1cli/internal/tui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	s = styles.Default()
)

func RunTUI(l *slog.Logger, done chan error) error {
	m := newModel(l, done)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return err
	}

	return nil
}

// Model represents the current state of the Bubbletea Application. It holds all of the data
// required to render the TUI.
type Model struct {
	// meta data
	logger    *slog.Logger
	isLoading bool
	// bubbles
	spinner spinner.Model
	// window size
	width  int
	height int
	// channels
	done chan error // this channel closes with an exit code when the TUI has exited (any non zero code indicates an error)
	// model data
	drivers map[uint8]domain.Driver
	event   domain.RaceWeekendEvent
}

// Init return the initial command for the TUI to run.
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles incoming messages and update the model accordingly.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.logger.Debug("received tea.Msg", "type", fmt.Sprintf("%T", msg))

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return handleKeyMsg(m, msg)
	case tea.WindowSizeMsg:
		return handleWindowSizeMsg(m, msg)
	case RaceWeekendEventMsg:
		m.isLoading = false
		m.event = msg.data
		return m, nil
	case DriversMsg:
		m.isLoading = false
		m.drivers = msg.data
		return m, nil
	default:
		var cmd tea.Cmd
		if m.isLoading {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		return m, cmd
	}
}

// View renders the TUI view as a string for the terminal.
func (m Model) View() string {
	v := ""

	if m.isLoading {
		v = m.spinner.View() + " loading..."
	} else {
		v = lipgloss.JoinVertical(
			lipgloss.Center,
			viewHeader(m),
			viewPadding(m),
			viewTable(m),
		)
	}

	return s.Doc.Width(m.width).Render(v)
}

/* Tea Messages
------------------------------------------------------------------------------------------------- */

type RaceWeekendEventMsg struct {
	data domain.RaceWeekendEvent
}

type DriversMsg struct {
	data map[uint8]domain.Driver
}

// // DriverInfoMsg represents intrinsic data about a driver as well as updates to live-timing data
// // like grid position, gaps, etc.
// type DriverInfoMsg struct {
// 	Number       int
// 	ShortName    string
// 	Name         string
// 	Position     int
// 	IntervalGap  string
// 	LeaderGap    string
// 	TireCompound string
// 	TireLapCount int
// 	LastLapTime  string
// 	BestLapTime  string
// 	InPit        *bool
// }

// // LapCountMsg represents the total number of planned laps in a session that is lap-limited, along
// // with the current lap count of the lead car.
// type LapCountMsg struct {
// 	Total     int
// 	Completed int
// }

// // raceCtrlMsg represents communication sent out from race control which includes things like
// // safety car alerts, debris on track, and information from the stewards.
// type RaceCtrlMsg struct {
// 	Category string
// 	Message  string
// }

/* Tea Message Handlers
------------------------------------------------------------------------------------------------- */

// handleKeyMsg is a tea.Msg handler that handles key press messages including ctrl+c and q to quit
// the TUI application.
func handleKeyMsg(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.logger.Debug("closing done channel and exiting ")
		close(m.done)
		return m, tea.Quit
	}
	return m, nil
}

// handleWindowSizeMsg is a tea.Msg handler that handles window resize events and stores the current
// window size of the terminal in the tea model.
func handleWindowSizeMsg(m Model, msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	h, v := s.Doc.GetFrameSize()
	m.width = msg.Width - h
	m.height = msg.Height - v
	return m, nil
}

// handleEventInfoMsg is a tea.Msg handler that handles incoming session info messages to set
// intrinsic data about the session along with the current status of the sessin and track
// func handleEventInfoMsg(m Model, msg EventInfoMsg) (Model, tea.Cmd) {
// 	// handle intrinsic data
// 	if msg.MeetingName != "" {
// 		m.eventInfo.MeetingName = msg.MeetingName
// 	}
// 	if msg.SessionName != "" {
// 		m.eventInfo.SessionName = msg.SessionName
// 	}
// 	switch msg.SessionType {
// 	case "Practice":
// 		m.eventInfo.SessionType = sessionTypePractice
// 	case "Qualifying":
// 		m.eventInfo.SessionType = sessionTypeQualifying
// 	case "Race":
// 		m.eventInfo.SessionType = sessionTypeRace
// 	default:
// 		m.eventInfo.SessionType = sessionTypeUnknown
// 	}
// 	// handle session status
// 	switch msg.SessionStatus {
// 	case "InProgress":
// 		m.eventInfo.SessionStatus = sessionStatusInProgress
// 	case "Completed":
// 		m.eventInfo.SessionStatus = sessionStatusCompleted
// 	case "":
// 		break // don't overwrite session status if the incoming message has an empty status
// 	default:
// 		m.eventInfo.SessionStatus = sessionStatusUnknown
// 	}
// 	// handle track status
// 	switch msg.TrackStatus {
// 	case "AllClear":
// 		m.eventInfo.TrackStatus = trackStatusAllClear
// 	case "Yellow":
// 		m.eventInfo.TrackStatus = trackStatusYellow
// 	case "VSCDeployed":
// 		m.eventInfo.TrackStatus = trackStatusVSCDeployed
// 	case "SCDeployed":
// 		m.eventInfo.TrackStatus = trackStatusSCDeployed
// 	case "Red":
// 		m.eventInfo.TrackStatus = trackStatusRed
// 	case "":
// 		break // don't overwrite track status if incoming message has an empty status
// 	default:
// 		m.eventInfo.TrackStatus = trackStatusUnknown
// 	}

// 	return m, nil
// }

// handleDriverInfoMsg is a tea.Msg handler that handles incoming driver info messages to set
// intrinsic data about the drivers in the session along with updates to live-timing data like
// grid position, gaps, etc.
// func handleDriverInfoMsg(m Model, msg DriverInfoMsg) (Model, tea.Cmd) {
// 	delta := driver{
// 		Number:      msg.Number,
// 		Name:        msg.Name,
// 		ShortName:   msg.ShortName,
// 		Position:    msg.Position,
// 		IntervalGap: msg.IntervalGap,
// 		LeaderGap:   msg.LeaderGap,
// 		LastLapTime: msg.LastLapTime,
// 		BestLapTime: msg.BestLapTime,
// 		InPit:       msg.InPit,
// 		Tire: tire{
// 			Compound: msg.TireCompound,
// 			LapCount: msg.TireLapCount,
// 		},
// 	}
// 	// Sometimes last lap updates before personal best lap; we'll can keep track of best lap times
// 	// for this scenario and correct it ourselves.
// 	if delta.BestLapTime == "" || (delta.LastLapTime != "" && delta.LastLapTime < delta.BestLapTime) {
// 		delta.BestLapTime = delta.LastLapTime
// 	}
// 	// Update/Set the driver data in the drivers list
// 	existingDriverInfo, ok := m.drivers[msg.Number]
// 	if ok {
// 		// existing driver entry must be merged, and the map updated
// 		m.drivers[msg.Number] = mergeDriverInfo(existingDriverInfo, delta)
// 	} else {
// 		// new driver entry can be safely added to the map without any fuss
// 		m.drivers[msg.Number] = delta
// 	}
// 	// update overall fastest lap data
// 	m = setFastestLap(m, m.drivers[msg.Number])

// 	return m, nil
// }

// handleLapCountMsg is a tea.Msg handler that handles incoming lap count messages to set the total
// planned number of laps along with the total number of completed laps of the lead car. Ignore
// zero values for total and completed.
// func handleLapCountMsg(m Model, msg LapCountMsg) (Model, tea.Cmd) {
// 	if msg.Total > 0 {
// 		m.totalPlannedLaps = msg.Total
// 	}
// 	if msg.Completed > 0 {
// 		m.completedLaps = msg.Completed
// 	}
// 	return m, nil
// }

// handleRaceCtrlMsg is a tea.Msg handler that handles incoming race control messages, converting
// them into a toast-style message stored in the tea model.
// func handleRaceCtrlMsg(m Model, msg RaceCtrlMsg) (Model, tea.Cmd) {
// 	m.raceCtrlMsg.Title = msg.Category
// 	m.raceCtrlMsg.Body = msg.Message
// 	m.raceCtrlMsg.RecevedAt = time.Now().Format(time.RFC3339)

// 	return m, nil
// }

/* Private State Helper Functions
------------------------------------------------------------------------------------------------- */

// newModel creates a new instance of the underlying TUI model.
func newModel(logger *slog.Logger, done chan error) Model {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot

	m := Model{
		logger:    logger,
		isLoading: true,
		spinner:   sp,
		drivers:   make(map[uint8]domain.Driver),
		done:      done,
	}

	return m
}

// mergeDriverInfo is a helper function to 'intelligently' merge data from a driver info message
// into existing driver info stored within the model.
// func mergeDriverInfo(d driver, delta driver) driver {
// 	newDriver := d

// 	if delta.Position > 0 {
// 		newDriver.Position = delta.Position
// 	}
// 	if delta.InPit != nil {
// 		newDriver.InPit = delta.InPit
// 	}
// 	// interval
// 	if delta.IntervalGap != "" {
// 		newDriver.IntervalGap = delta.IntervalGap
// 	}
// 	if delta.LeaderGap != "" {
// 		newDriver.LeaderGap = delta.LeaderGap
// 	}
// 	// lap times
// 	if newDriver.BestLapTime == "" || (delta.BestLapTime < newDriver.BestLapTime) {
// 		newDriver.BestLapTime = delta.BestLapTime
// 	}
// 	if delta.LastLapTime != "" {
// 		newDriver.LastLapTime = delta.LastLapTime
// 	}
// 	// Tire delta
// 	if delta.Tire.Compound != "" {
// 		newDriver.Tire.Compound = delta.Tire.Compound
// 	}
// 	if delta.Tire.LapCount > 0 {
// 		newDriver.Tire.LapCount = delta.Tire.LapCount
// 	}

// 	return newDriver
// }

// setFastestLap sets the overall fastest lap and driver-specific fastest lap, based on the personal
// fastest lap times and last lap times of the driver info message.
// func setFastestLap(m Model, d driver) Model {
// 	if m.fastestLapTime == "" || (d.BestLapTime < m.fastestLapTime) {
// 		m.fastestLapTime = d.BestLapTime
// 		m.fastestLapOwner = d.Number
// 	}

// 	return m
// }

/* Private View Helper Functions
------------------------------------------------------------------------------------------------- */

// getPadding returns the padding view component
func viewPadding(m Model) string {
	return lipgloss.PlaceHorizontal(
		m.width,
		lipgloss.Center,
		"",
		lipgloss.WithWhitespaceChars("."),
		lipgloss.WithWhitespaceForeground(s.Color.Subtle),
	)
}

// viewHeader returns the header view component
func viewHeader(m Model) string {
	titleBarStyle := s.TitleBar
	subtitleBarStyle := s.SubtitleBar
	padding := viewPadding(m)

	subtitleContent := m.event.Name
	if m.event.Session.Type == domain.SessionTypeRace {
		subtitleContent = fmt.Sprintf("Race: %d / %d Laps", m.event.Session.CurrentLap, m.event.Session.TotalLaps)
	}

	// todo: track status
	// switch m.eventInfo.TrackStatus {
	// case trackStatusAllClear:
	// 	break
	// case trackStatusYellow:
	// 	subtitleContent = subtitleContent + s.Yellow.Render(" - Yellow Flag is Out")
	// case trackStatusVSCDeployed:
	// 	subtitleContent = subtitleContent + s.Yellow.Render(" - VSC Deployed")
	// case trackStatusSCDeployed:
	// 	subtitleContent = subtitleContent + s.Yellow.Render(" - Safety Car Deployed")
	// case trackStatusRed:
	// 	subtitleContent = subtitleContent + s.Red.Render(" - Session Red Flagged")
	// }

	return lipgloss.JoinVertical(
		lipgloss.Center,
		titleBarStyle.Width(m.width).Render(m.event.FullName),
		padding,
		padding,
		subtitleBarStyle.Width(m.width).Render(subtitleContent),
	)
}

// viewTable returns the table view component
func viewTable(m Model) string {
	baseStyle := lipgloss.NewStyle().Padding(0, 1, 1, 1)
	drivers := sortDrivers(m.drivers)
	rows := make([][]string, 0, len(drivers))

	for _, d := range drivers {
		rows = append(rows, []string{
			driverPosition(d),
			driverName(d, m.event),
			driverIntervalGap(d),
			driverLeaderGap(d),
			driverTire(d), // tire/stint data
			"TODO",        // sectors
			driverLastLap(d, &m.event),
			driverBestLap(d, &m.event), // best
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		// BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := baseStyle

			if row == len(rows)-1 {
				style = style.Padding(0, 1)
			}

			if col == 0 {
				style = style.Align(lipgloss.Right)
			}

			return style
		}).
		Headers("POS", "DRIVER", "INT", "LEADER", "TIRE", "SECTORS", "LAST", "BEST").
		Rows(rows...)

	return t.Render()
}

// sortDrivers returns a sorted of list of drivers, sorted by their leaderboard position in the
// session used by the timing table.
func sortDrivers(driverMap map[uint8]domain.Driver) []domain.Driver {
	drivers := make([]domain.Driver, 0, len(driverMap))
	for _, driver := range driverMap {
		drivers = append(drivers, driver)
	}

	sort.Slice(drivers, func(i, j int) bool {
		return drivers[i].Position < drivers[j].Position
	})

	return drivers
}

// driverPosition returns a sanitized version of the driver's position on the leaderboard
func driverPosition(d domain.Driver) string {
	v := "-"
	if pos := strconv.Itoa(int(d.Position)); pos != "0" {
		v = pos
	}
	return v
}

// driverName returns the driver name formatted with the team color and fastsest lap indicator when
// appropriate formatted for the timing table
func driverName(d domain.Driver, e domain.RaceWeekendEvent) string {
	c := lipgloss.Color("#" + d.TeamColor)
	n := lipgloss.NewStyle().Foreground(c).Render("▍") + d.ShortName
	if d.Number == e.Session.FastestLapOwner {
		n = n + s.Purple.Render(" ⏱")
	}
	return n
}

var (
	leaderRe = regexp.MustCompile(`/LAP/`)
)

// driverIntervalGap returns the driver interval to the car ahead formatted for the timing table.
func driverIntervalGap(d domain.Driver) string {
	if d.IntervalGap == "" || d.OutOfSession || leaderRe.MatchString(d.IntervalGap) {
		return "-"
	}
	return d.IntervalGap
}

// driverLeaderGap returns the driver interval to the leader car formatted for the timing table.
func driverLeaderGap(d domain.Driver) string {
	if d.LeaderGap == "" || d.OutOfSession || leaderRe.MatchString(d.LeaderGap) {
		return "-"
	}
	return d.LeaderGap
}

// driverLeaderGap returns the driver interval to the leader car formatted for the timing table.
func driverTire(d domain.Driver) string {
	if d.TireCompound == "" || d.OutOfSession {
		return "-"
	}
	t := d.TireCompound[:1]
	tireStyle := lipgloss.NewStyle()
	switch d.TireCompound {
	case domain.TireCompoundSoft:
		tireStyle = tireStyle.Foreground(s.Color.SoftTire)
	case domain.TireCompoundMedium:
		tireStyle = tireStyle.Foreground(s.Color.MediumTire)
	case domain.TireCompoundIntermediate:
		tireStyle = tireStyle.Foreground(s.Color.IntermediateTire)
	case domain.TireCompoundFullWet:
		tireStyle = tireStyle.Foreground(s.Color.WetTire)
	case domain.TireCompoundUnknown:
		t = "X"
	}

	return fmt.Sprintf("%s %d Laps", tireStyle.Render(string(t)), d.TireLapCount)
}

func driverLastLap(d domain.Driver, e *domain.RaceWeekendEvent) string {
	v := "-"

	if d.LastLap.Time != "" {
		v = d.LastLap.Time

		if e.Session.FastestLapTime == "" || d.LastLap.Time < e.Session.FastestLapTime {
			v = lipgloss.NewStyle().Foreground(s.Color.Purple).Render(v)
			e.Session.FastestLapTime = d.LastLap.Time
		} else if d.LastLap.IsPersonalBest {
			v = lipgloss.NewStyle().Foreground(s.Color.Green).Render(v)
		} else {
			v = lipgloss.NewStyle().Foreground(s.Color.Yellow).Render(v)
		}
	}

	return v
}

func driverBestLap(d domain.Driver, e *domain.RaceWeekendEvent) string {
	v := "-"

	if d.BestLapTime != "" {
		v = d.BestLapTime

		if d.BestLapTime <= e.Session.FastestLapTime {
			v = lipgloss.NewStyle().Foreground(lipgloss.Color(s.Color.Purple)).Render(v)
			e.Session.FastestLapTime = d.BestLapTime
		}
	}

	return v
}
