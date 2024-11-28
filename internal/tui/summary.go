package summary

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/bcdxn/f1cli/internal/tui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	s = styles.DefaultStyles()
)

func RunTUI(l *slog.Logger, done chan int) error {
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
	done chan int // this channel closes with an exit code when the TUI has exited (any non zero code indicates an error)
	// model data
	eventInfo        eventInfo
	drivers          map[int]driver
	fastestLapTime   string
	fastestLapOwner  int
	raceCtrlMsg      raceCtrlMsgToast
	totalPlannedLaps int
	completedLaps    int
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
	case EventInfoMsg:
		return handleEventInfoMsg(m, msg)
	case DriverInfoMsg:
		return handleDriverInfoMsg(m, msg)
	case LapCountMsg:
		return handleLapCountMsg(m, msg)
	case RaceCtrlMsg:
		return handleRaceCtrlMsg(m, msg)
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
		)
	}

	return s.Doc.Width(m.width).Render(v)
}

/* Tea Messages
------------------------------------------------------------------------------------------------- */

// EventInfoMsg represents a message that passes intrinsic data about the race weekend; although
// there is no limit, typically this message only sent once as it should not change during the
// course of a session.
type EventInfoMsg struct {
	MeetingName   string // The name of the entire event, e.g. Las Vegas Grand Prix
	SessionType   string // The type of session, e.g.: Qualifying
	SessionName   string // The name of the specific session within the event, e.g.: Practice 3
	SessionStatus string // Started, Finished, Inactive, Finalised, Ends
	TrackStatus   string // AllClear, Yellow, VSCDeployed, SCDeployed, Red
}

// DriverInfoMsg represents intrinsic data about a driver as well as updates to live-timing data
// like grid position, gaps, etc.
type DriverInfoMsg struct {
	Number       int
	ShortName    string
	Name         string
	Position     int
	IntervalGap  string
	LeaderGap    string
	TireCompound string
	TireLapCount int
	LastLapTime  string
	BestLapTime  string
	InPit        *bool
}

// LapCountMsg represents the total number of planned laps in a session that is lap-limited, along
// with the current lap count of the lead car.
type LapCountMsg struct {
	Total     int
	Completed int
}

// raceCtrlMsg represents communication sent out from race control which includes things like
// safety car alerts, debris on track, and information from the stewards.
type RaceCtrlMsg struct {
	Category string
	Message  string
}

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
func handleEventInfoMsg(m Model, msg EventInfoMsg) (Model, tea.Cmd) {
	// handle intrinsic data
	if msg.MeetingName != "" {
		m.eventInfo.MeetingName = msg.MeetingName
	}
	if msg.SessionName != "" {
		m.eventInfo.SessionName = msg.SessionName
	}
	switch msg.SessionType {
	case "Practice":
		m.eventInfo.SessionType = sessionTypePractice
	case "Qualifying":
		m.eventInfo.SessionType = sessionTypeQualifying
	case "Race":
		m.eventInfo.SessionType = sessionTypeRace
	default:
		m.eventInfo.SessionType = sessionTypeUnknown
	}
	// handle session status
	switch msg.SessionStatus {
	case "InProgress":
		m.eventInfo.SessionStatus = sessionStatusInProgress
	case "Completed":
		m.eventInfo.SessionStatus = sessionStatusCompleted
	case "":
		break // don't overwrite session status if the incoming message has an empty status
	default:
		m.eventInfo.SessionStatus = sessionStatusUnknown
	}
	// handle track status
	switch msg.TrackStatus {
	case "AllClear":
		m.eventInfo.TrackStatus = trackStatusAllClear
	case "Yellow":
		m.eventInfo.TrackStatus = trackStatusYellow
	case "VSCDeployed":
		m.eventInfo.TrackStatus = trackStatusVSCDeployed
	case "SCDeployed":
		m.eventInfo.TrackStatus = trackStatusSCDeployed
	case "Red":
		m.eventInfo.TrackStatus = trackStatusRed
	case "":
		break // don't overwrite track status if incoming message has an empty status
	default:
		m.eventInfo.TrackStatus = trackStatusUnknown
	}

	return m, nil
}

// handleDriverInfoMsg is a tea.Msg handler that handles incoming driver info messages to set
// intrinsic data about the drivers in the session along with updates to live-timing data like
// grid position, gaps, etc.
func handleDriverInfoMsg(m Model, msg DriverInfoMsg) (Model, tea.Cmd) {
	delta := driver{
		Number:      msg.Number,
		Name:        msg.Name,
		ShortName:   msg.ShortName,
		Position:    msg.Position,
		IntervalGap: msg.IntervalGap,
		LeaderGap:   msg.LeaderGap,
		LastLapTime: msg.LastLapTime,
		BestLapTime: msg.BestLapTime,
		InPit:       msg.InPit,
		Tire: tire{
			Compound: msg.TireCompound,
			LapCount: msg.TireLapCount,
		},
	}
	// Sometimes last lap updates before personal best lap; we'll can keep track of best lap times
	// for this scenario and correct it ourselves.
	if delta.BestLapTime == "" || (delta.LastLapTime != "" && delta.LastLapTime < delta.BestLapTime) {
		delta.BestLapTime = delta.LastLapTime
	}
	// Update/Set the driver data in the drivers list
	existingDriverInfo, ok := m.drivers[msg.Number]
	if ok {
		// existing driver entry must be merged, and the map updated
		m.drivers[msg.Number] = mergeDriverInfo(existingDriverInfo, delta)
	} else {
		// new driver entry can be safely added to the map without any fuss
		m.drivers[msg.Number] = delta
	}
	// update overall fastest lap data
	m = setFastestLap(m, m.drivers[msg.Number])

	return m, nil
}

// handleLapCountMsg is a tea.Msg handler that handles incoming lap count messages to set the total
// planned number of laps along with the total number of completed laps of the lead car. Ignore
// zero values for total and completed.
func handleLapCountMsg(m Model, msg LapCountMsg) (Model, tea.Cmd) {
	if msg.Total > 0 {
		m.totalPlannedLaps = msg.Total
	}
	if msg.Completed > 0 {
		m.completedLaps = msg.Completed
	}
	return m, nil
}

// handleRaceCtrlMsg is a tea.Msg handler that handles incoming race control messages, converting
// them into a toast-style message stored in the tea model.
func handleRaceCtrlMsg(m Model, msg RaceCtrlMsg) (Model, tea.Cmd) {
	m.raceCtrlMsg.Title = msg.Category
	m.raceCtrlMsg.Body = msg.Message
	m.raceCtrlMsg.RecevedAt = time.Now().Format(time.RFC3339)

	return m, nil
}

/* Private State Helper Functions
------------------------------------------------------------------------------------------------- */

// newModel creates a new instance of the underlying TUI model.
func newModel(logger *slog.Logger, done chan int) Model {
	drivers := make(map[int]driver)
	loading := "loading..."
	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return Model{
		logger:    logger,
		isLoading: true,
		spinner:   s,
		drivers:   drivers,
		done:      done,
		eventInfo: eventInfo{
			MeetingName:   loading,
			SessionName:   loading,
			SessionType:   sessionTypeUnknown,
			SessionStatus: sessionStatusUnknown,
			TrackStatus:   trackStatusUnknown,
		},
	}
}

// mergeDriverInfo is a helper function to 'intelligently' merge data from a driver info message
// into existing driver info stored within the model.
func mergeDriverInfo(d driver, delta driver) driver {
	newDriver := d

	if delta.Position > 0 {
		newDriver.Position = delta.Position
	}
	if delta.InPit != nil {
		newDriver.InPit = delta.InPit
	}
	// interval
	if delta.IntervalGap != "" {
		newDriver.IntervalGap = delta.IntervalGap
	}
	if delta.LeaderGap != "" {
		newDriver.LeaderGap = delta.LeaderGap
	}
	// lap times
	if newDriver.BestLapTime == "" || (delta.BestLapTime < newDriver.BestLapTime) {
		newDriver.BestLapTime = delta.BestLapTime
	}
	if delta.LastLapTime != "" {
		newDriver.LastLapTime = delta.LastLapTime
	}
	// Tire delta
	if delta.Tire.Compound != "" {
		newDriver.Tire.Compound = delta.Tire.Compound
	}
	if delta.Tire.LapCount > 0 {
		newDriver.Tire.LapCount = delta.Tire.LapCount
	}

	return newDriver
}

// setFastestLap sets the overall fastest lap and driver-specific fastest lap, based on the personal
// fastest lap times and last lap times of the driver info message.
func setFastestLap(m Model, d driver) Model {
	if m.fastestLapTime == "" || (d.BestLapTime < m.fastestLapTime) {
		m.fastestLapTime = d.BestLapTime
		m.fastestLapOwner = d.Number
	}

	return m
}

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

	subtitleContent := m.eventInfo.SessionName
	if subtitleContent == "Race" {
		subtitleContent = fmt.Sprintf("Race: %d / %d Laps", m.completedLaps, m.totalPlannedLaps)
	}

	switch m.eventInfo.TrackStatus {
	case trackStatusAllClear:
		break
	case trackStatusYellow:
		subtitleContent = subtitleContent + s.Yellow.Render(" - Yellow Flag is Out")
	case trackStatusVSCDeployed:
		subtitleContent = subtitleContent + s.Yellow.Render(" - VSC Deployed")
	case trackStatusSCDeployed:
		subtitleContent = subtitleContent + s.Yellow.Render(" - Safety Car Deployed")
	case trackStatusRed:
		subtitleContent = subtitleContent + s.Red.Render(" - Session Red Flagged")
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		titleBarStyle.Width(m.width).Render(m.eventInfo.MeetingName),
		padding,
		padding,
		subtitleBarStyle.Width(m.width).Render(subtitleContent),
	)
}

/* Private Types
------------------------------------------------------------------------------------------------- */

type sessionType int

const (
	sessionTypeUnknown sessionType = iota
	sessionTypePractice
	sessionTypeQualifying
	sessionTypeRace
)

// Session Status
type sessionStatus int

const (
	sessionStatusUnknown sessionStatus = iota
	sessionStatusInProgress
	sessionStatusCompleted
)

// Track status enumerations
type trackStatus int

const (
	trackStatusUnknown trackStatus = iota
	trackStatusAllClear
	trackStatusYellow
	trackStatusVSCDeployed
	trackStatusSCDeployed
	trackStatusRed
)

// eventInfo represents intrinsic data about the race weekend.
type eventInfo struct {
	MeetingName   string
	SessionName   string
	SessionType   sessionType
	SessionStatus sessionStatus
	TrackStatus   trackStatus
}

// driver represents intrinsic data about the driver like name, as well as live-timing data like
// position and interval data.
type driver struct {
	Number      int
	ShortName   string
	Name        string
	Position    int
	IntervalGap string
	LeaderGap   string
	Tire        tire
	LastLapTime string
	BestLapTime string
	InPit       *bool
}

// tire represents tire data for a specific car.
type tire struct {
	Compound string
	LapCount int
}

// raceCtrlMsg represents communication sent out from race control which includes things like
// safety car alerts, debris on track, and information from the stewards.
type raceCtrlMsgToast struct {
	Title     string
	Body      string
	RecevedAt string
}
