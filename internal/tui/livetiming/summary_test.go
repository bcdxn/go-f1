package summary

import (
	"io"
	"log/slog"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// func init() {
// 	lipgloss.SetColorProfile(termenv.Ascii)
// }

// Test that the app exits after receiving a ctrl+c key press event and notifies its exit via the
// provided 'done' channel.
func TestAppExitOnCtrlC(t *testing.T) {
	done := make(chan int)
	tm := teatest.NewTestModel(
		t,
		New(testLogger(t), done),
		teatest.WithInitialTermSize(100, 150),
	)

	code, m := exitTeaProgram(t, tm, done)

	if code != 0 {
		t.Fatal("should not have exited with error but found exit code", code)
	}
	if !m.isLoadingReference {
		t.Fatal("m.isLoadingReference should be true")
	}
}

// TestHandleEventInfoMsg validates that the EventInfoMsg is handled correctly
func TestHandleEventInfoMsg(t *testing.T) {
	done := make(chan int)
	m := New(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	// create a new EventInfoMsg for testing
	msg := EventInfoMsg{
		MeetingName: "Test Meeting",
		SessionType: "Practice",
		SessionName: "Practice 1",
	}
	// send the message to the TUI app
	tm.Send(msg)
	// stop the program so we can look at the final model state
	_, fm := exitTeaProgram(t, tm, done)
	// check that the message was handled properly
	if fm.eventInfo.MeetingName != msg.MeetingName {
		t.Fatalf("expected %s but found %s", msg.MeetingName, fm.eventInfo.MeetingName)
	}
	if fm.eventInfo.SessionType != sessionTypePractice {
		t.Fatalf("expected %d but found %d", sessionTypePractice, fm.eventInfo.SessionType)
	}
}

func TestHandleDriverInfoMsg(t *testing.T) {
	done := make(chan int)
	m := New(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	// create a new DriverInfoMsg for testing
	msg := DriverInfoMsg{
		Number:       44,
		Name:         "Lewis Hamilton",
		ShortName:    "HAM",
		Position:     1,
		IntervalGap:  "LAP 1",
		LeaderGap:    "LAP 1",
		LastLapTime:  "1.11:111",
		BestLapTime:  "1.11:222",
		InPit:        boolPointer(false),
		TireCompound: "SOFT",
		TireLapCount: 1,
	}
	// send the message to the TUI app
	tm.Send(msg)
	// stop the program so we can look at the final model state
	_, fm := exitTeaProgram(t, tm, done)
	// check that the message was handled properly
	if fm.drivers[44].Name != msg.Name {
		t.Fatalf("expected %s but found %s", msg.Name, fm.drivers[44].Name)
	}
	if fm.drivers[44].ShortName != msg.ShortName {
		t.Fatalf("expected %s but found %s", msg.Name, fm.drivers[44].Name)
	}
	if fm.drivers[44].Position != msg.Position {
		t.Fatalf("expected %d but found %d", msg.Position, fm.drivers[44].Position)
	}
}

func TestHandleDriverInfoMsgDelta(t *testing.T) {
	done := make(chan int)
	m := New(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	// create a new DriverInfoMsg for testing
	tm.Send(DriverInfoMsg{
		Number:       44,
		Name:         "Lewis Hamilton",
		ShortName:    "HAM",
		Position:     1,
		IntervalGap:  "LAP 1",
		LeaderGap:    "LAP 1",
		LastLapTime:  "1.11:111",
		BestLapTime:  "1.11:222",
		InPit:        boolPointer(false),
		TireCompound: "SOFT",
		TireLapCount: 1,
	})
	msg := DriverInfoMsg{
		Number:       44,
		Name:         "Lewis Hamilton",
		ShortName:    "HAM",
		Position:     5,
		IntervalGap:  "LAP 1",
		LeaderGap:    "LAP 1",
		LastLapTime:  "1.11:111",
		BestLapTime:  "1.11:222",
		InPit:        boolPointer(false),
		TireCompound: "SOFT",
		TireLapCount: 1,
	}
	msg2 := DriverInfoMsg{
		Number:       44,
		Position:     0,
		InPit:        nil,
		TireLapCount: 5,
	}
	// send the delta message to the TUI app
	tm.Send(msg)
	// send the delta message to the TUI app
	tm.Send(msg2)
	// stop the program so we can look at the final model state
	_, fm := exitTeaProgram(t, tm, done)
	if fm.drivers[44].Position != msg.Position {
		// Ensure that a zero value doesn't overwrite existing value
		t.Fatalf("expected %d but found %d", msg.Position, fm.drivers[44].Position)
	}
	if fm.drivers[44].Tire.Compound != msg.TireCompound {
		// Ensure that a zero value doesn't overwrite existing value
		t.Fatalf("expected %s but found %s", msg.TireCompound, fm.drivers[44].Tire.Compound)
	}
	if fm.drivers[44].Tire.LapCount != msg2.TireLapCount {
		// Ensure that lap count is updated
		t.Fatalf("expected %d but found %d", msg2.TireLapCount, fm.drivers[44].Tire.LapCount)
	}
	if fm.drivers[44].InPit != msg.InPit {
		// Ensure that a zero value doesn't overwrite existing value
		t.Fatalf("expected %b but found %b", msg.InPit, fm.drivers[44].InPit)
	}
}

func TestHandleDriverInfoMsgFastestLap(t *testing.T) {
	done := make(chan int)
	m := New(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	msg := DriverInfoMsg{
		Number:       44,
		Name:         "Lewis Hamilton",
		ShortName:    "HAM",
		Position:     5,
		IntervalGap:  "LAP 1",
		LeaderGap:    "LAP 1",
		LastLapTime:  "1.11:222",
		BestLapTime:  "1.11:111",
		InPit:        boolPointer(false),
		TireCompound: "SOFT",
		TireLapCount: 1,
	}
	// send the message to the TUI app
	tm.Send(msg)
	// stop the program so we can look at the final model state
	_, fm := exitTeaProgram(t, tm, done)
	if fm.fastestLapTime != msg.BestLapTime {
		t.Fatalf("expected %s but found %s", msg.BestLapTime, fm.fastestLapTime)
	}
	if fm.fastestLapOwner != 44 {
		t.Fatalf("expected %d but found %d", 44, fm.fastestLapOwner)
	}
}

func TestHandleDriverInfoMsgFastestLapUpdate(t *testing.T) {
	done := make(chan int)
	m := New(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	msg := DriverInfoMsg{
		Number:       44,
		Name:         "Lewis Hamilton",
		ShortName:    "HAM",
		Position:     5,
		IntervalGap:  "LAP 1",
		LeaderGap:    "LAP 1",
		LastLapTime:  "1.11:111",
		BestLapTime:  "1.11:222",
		InPit:        boolPointer(false),
		TireCompound: "SOFT",
		TireLapCount: 1,
	}
	// send the message to the TUI app
	tm.Send(msg)

	msg = DriverInfoMsg{
		Number:       1,
		Name:         "Max Verstappen",
		ShortName:    "VER",
		Position:     6,
		IntervalGap:  "LAP 1",
		LeaderGap:    "LAP 1",
		LastLapTime:  "1.11:000",
		BestLapTime:  "1.12:000",
		InPit:        boolPointer(false),
		TireCompound: "SOFT",
		TireLapCount: 1,
	}
	// send the delta message to the TUI app
	tm.Send(msg)
	// stop the program so we can look at the final model state
	_, fm := exitTeaProgram(t, tm, done)
	// make assertions
	if fm.fastestLapTime != msg.LastLapTime {
		t.Fatalf("expected %s but found %s", msg.LastLapTime, fm.fastestLapTime)
	}
	if fm.fastestLapOwner != 1 {
		t.Fatalf("expected %d but found %d", 1, fm.fastestLapOwner)
	}
}

func TestHandleLapCountMsg(t *testing.T) {
	done := make(chan int)
	m := New(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	msg := LapCountMsg{
		Total:     51,
		Completed: 27,
	}
	// send the message to the TUI app
	tm.Send(msg)

	msg2 := LapCountMsg{
		Completed: 28,
	}
	msg3 := LapCountMsg{
		Completed: 28,
	}
	// send the message to the TUI app
	tm.Send(msg)
	tm.Send(msg2)
	tm.Send(msg3)
	// stop the program so we can look at the final model state
	_, fm := exitTeaProgram(t, tm, done)
	// make assertions
	if fm.totalPlannedLaps != msg.Total {
		t.Fatalf("expected %d but found %d", msg.Total, fm.totalPlannedLaps)
	}
	if fm.completedLaps != msg2.Completed {
		t.Fatalf("expected %d but found %d", msg2.Completed, fm.completedLaps)
	}
}

func TestHandleRaceCtrlMsg(t *testing.T) {
	done := make(chan int)
	m := New(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	msg := RaceCtrlMsg{
		Category: "Yellow Flag",
		Message:  "Race Control - Yellow flag in Sector 2",
	}
	// send the message to the TUI app
	tm.Send(msg)
	// stop the program so we can look at the final model state
	_, fm := exitTeaProgram(t, tm, done)
	// make assertions
	if fm.raceCtrlMsg.Title != msg.Category {
		t.Fatalf("expected %s but found %s", msg.Category, fm.raceCtrlMsg.Title)
	}
	if fm.raceCtrlMsg.Body != msg.Message {
		t.Fatalf("expected %s but found %s", msg.Message, fm.raceCtrlMsg.Body)
	}
}

func TestWindowInitialSize(t *testing.T) {
	done := make(chan int)
	m := New(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	// stop the program so we can look at the final model state
	_, fm := exitTeaProgram(t, tm, done)
	if fm.width != 98 {
		t.Fatalf("expected %d but found %d", 98, fm.width)
	}
	if fm.height != 148 {
		t.Fatalf("expected %d but found %d", 148, fm.width)
	}
}

func TestInitialView(t *testing.T) {
	done := make(chan int)
	m := New(testLogger(t), done)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(100, 150),
	)
	exitTeaProgram(t, tm, done)
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

// exitTeaProgram is a test helper function that sends a keypress event to quit the bubbletea test
// program, waits for it exit, and then returns the exit code along with the final model state
func exitTeaProgram(t *testing.T, tm *teatest.TestModel, done chan int) (int, Model) {
	var code int
	t.Helper()
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	for {
		// listen for exit code on the on done channel
		c, ok := <-done
		code = c
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

	return code, m
}

func boolPointer(b bool) *bool {
	return &b
}
