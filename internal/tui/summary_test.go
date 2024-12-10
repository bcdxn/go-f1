package summary

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bcdxn/f1cli/internal/domain"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// func init() {
// 	lipgloss.SetColorProfile(termenv.Ascii)
// }

// Test that the app exits after receiving a ctrl+c key press event and notifies its exit via the
// provided 'done' channel.
func TestAppExitOnCtrlC(t *testing.T) {
	done := make(chan error)
	tm := teatest.NewTestModel(
		t,
		newModel(testLogger(t), done),
		teatest.WithInitialTermSize(100, 150),
	)

	m, err := exitTestTeaProgram(t, tm, done)

	if err != nil {
		t.Fatal("should not have exited with error - ", err)
	}
	if !m.isLoading {
		t.Fatal("m.isLoading should be true")
	}
}

// TestHandleEventInfoMsg validates that the EventInfoMsg is handled correctly
func TestHandleRaceWeekendEventMsg(t *testing.T) {
	done := make(chan error)
	m := newModel(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	msg := RaceWeekendEventMsg{
		Data: domain.RaceWeekendEvent{
			Name:        "United States Grand Prix",
			FullName:    "FORMULA 1 PIRELLI UNITED STATES GRAND PRIX 2024",
			Location:    "Austin",
			RoundNumber: 19,
			CountryCode: "USA",
			Session: domain.Session{
				Type:       domain.SessionTypeRace,
				Name:       "Race",
				CurrentLap: 2,
				TotalLaps:  56,
			},
		},
	}
	tm.Send(msg)
	exitTestTeaProgram(t, tm, done)
	out, err := io.ReadAll(tm.FinalOutput(t))
	if err != nil {
		t.Error(err)
	}
	teatest.RequireEqualOutput(t, out)
}

// TestWindowInitialSize checks that the default starting model captures the terminal window size
// as those dimensions are required to render the view appropriately.
func TestWindowInitialSize(t *testing.T) {
	done := make(chan error)
	m := newModel(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	// stop the program so we can look at the final model state
	fm, _ := exitTestTeaProgram(t, tm, done)
	if fm.width != 98 {
		t.Fatalf("expected %d but found %d", 98, fm.width)
	}
	if fm.height != 148 {
		t.Fatalf("expected %d but found %d", 148, fm.width)
	}
}

// TestInitialView checks that the loading spinner is shown before any data is loaded from the F1
// LiveTiming API.
func TestInitialView(t *testing.T) {
	done := make(chan error)
	m := newInitialModel(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)

	exitTestTeaProgram(t, tm, done)
	// capture/check TUI view
	out, err := io.ReadAll(tm.FinalOutput(t))
	if err != nil {
		t.Error(err)
	}
	if !bytes.Contains(out, []byte("loading...")) {
		t.Errorf("expected view to contain 'loading...' but found '%s'", out)
	}
}

// TestRacePreStart checks that view is rendered correctly before any timing data has been loaded
// from the F1 LiveTiming API.
func TestRacePreStart(t *testing.T) {
	done := make(chan error)
	m := newIsLoadedRaceModel(t, testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	exitTestTeaProgram(t, tm, done)
	// capture/check TUI view
	out, err := io.ReadAll(tm.FinalOutput(t))
	if err != nil {
		t.Error(err)
	}
	teatest.RequireEqualOutput(t, out)
}

// TestRaceLapIncrement checks that the view is rendered correctly as lap count is incremented over
// the course of a race.
func TestRaceLapIncrement(t *testing.T) {
	done := make(chan error)
	m := newIsLoadedRaceModel(t, testLogger(t), done)
	rwEvent := m.event
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	rwEvent.Session.CurrentLap = 1
	tm.Send(RaceWeekendEventMsg{
		Data: rwEvent,
	})
	rwEvent.Session.CurrentLap = 2
	tm.Send(RaceWeekendEventMsg{
		Data: rwEvent,
	})
	exitTestTeaProgram(t, tm, done)
	// capture/check TUI view
	out, err := io.ReadAll(tm.FinalOutput(t))
	if err != nil {
		t.Error(err)
	}
	teatest.RequireEqualOutput(t, out)

	exitTestTeaProgram(t, tm, done)
}

func TestTimingTable(t *testing.T) {
	dl := driverList(t)
	e := raceWeekendEvent(t)

	done := make(chan error)
	m := newIsLoadedRaceModel(t, testLogger(t), done)
	m.drivers = dl
	m.event = e
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	exitTestTeaProgram(t, tm, done)
	// capture/check TUI view
	out, err := io.ReadAll(tm.FinalOutput(t))
	if err != nil {
		t.Error(err)
	}
	teatest.RequireEqualOutput(t, out)
}

/* Test Helper Functions
------------------------------------------------------------------------------------------------- */

// testLogger is a test helper function that creates a new instance of a logger that follows the
// slog interface and that writes all logs to /dev/null
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// exitTestTeaProgram is a test helper function that sends a keypress event to quit the bubbletea test
// program, waits for it exit, and then returns the error along with the final model state
func exitTestTeaProgram(t *testing.T, tm *teatest.TestModel, done chan error) (Model, error) {
	var err error
	t.Helper()
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	for {
		// listen for exit code on the on done channel
		e, ok := <-done
		err = e
		// when the done channel is closed, we will all the test to continue
		if !ok {
			break
		}
	}
	fm := tm.FinalModel(t)
	m, ok := fm.(Model)
	if !ok {
		t.Fatalf("final model have the wrong type: %T", fm)
	}

	return m, err
}

// intitialModel returns a default initial model as would be returned by the TUI outside of the
// testing environment.
func newInitialModel(logger *slog.Logger, done chan error) Model {
	return newModel(logger, done)
}

// newIsLoadedRaceModel returns a model that represents the state of the application _after_
// receiving race weekend data and driver data loaded.
func newIsLoadedRaceModel(t *testing.T, logger *slog.Logger, done chan error) Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	startTime, err := time.Parse("2006-01-02T15:04:05-0700", "2024-10-20T14:00:00-0700")
	if err != nil {
		t.Errorf("error parsing date string in test %v", err)
	}

	return Model{
		logger:    logger,
		isLoading: false,
		spinner:   s,
		drivers:   make(map[uint8]domain.Driver),
		done:      done,
		event: domain.RaceWeekendEvent{
			Name:        "United States Grand Prix",
			FullName:    "FORMULA 1 PIRELLI UNITED STATES GRAND PRIX 2024",
			Location:    "Austin",
			RoundNumber: 19,
			CountryCode: "USA",
			Session: domain.Session{
				Type:       domain.SessionTypeRace,
				Name:       "Race",
				CurrentLap: 0,
				TotalLaps:  56,
				StartDate:  startTime,
			},
		},
	}
}

// driverList parses the testdata file and returns a map of Driver domain objects.
func driverList(t *testing.T) map[uint8]domain.Driver {
	driversJson, err := os.ReadFile(path.Join(testdataDir(), "drivers.json"))
	if err != nil {
		t.Errorf("error loading driver list data for test - %s", err.Error())
	}

	var drivers map[uint8]domain.Driver
	err = json.Unmarshal(driversJson, &drivers)
	if err != nil {
		t.Errorf("error loading driver list data for test - %s", err.Error())
	}

	return drivers
}

// raceWeekendEvent parses the testdata file and returns a RaceWeekendEvent domain object.
func raceWeekendEvent(t *testing.T) domain.RaceWeekendEvent {
	eventJson, err := os.ReadFile(path.Join(testdataDir(), "raceweekendevent.json"))
	if err != nil {
		t.Errorf("error loading race weekend event data for test - %s", err.Error())
	}

	var event domain.RaceWeekendEvent
	err = json.Unmarshal(eventJson, &event)
	if err != nil {
		t.Errorf("error loading race weekend event data for test - %s", err.Error())
	}

	return event
}

// getTestdataDir gets the testdata directory path relative to the invocation of the tests.
func testdataDir() string {
	_, p, _, _ := runtime.Caller(0)
	return path.Join(filepath.Dir(p), "testdata")
}
