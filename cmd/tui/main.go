package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/bcdxn/f1cli/internal/f1livetiming"
	summary "github.com/bcdxn/f1cli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// Create a file handler
	file, err := os.Create("app.log")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Create a text handler that writes to the file
	handler := slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: slog.LevelError,
	})

	// Create a logger with the file handler
	logger := slog.New(handler)
	done := make(chan error)

	tp := summary.NewTUI(logger, done)

	go func() {
		err := summary.RunTUI(tp)
		if err != nil {
			log.Fatal("Error starting TUI:", err.Error())
		}
	}()

	// Start Client
	client := f1livetiming.NewClient(f1livetiming.WithLogger(logger))
	go func() {
		client.Negotiate()
		client.Connect()
	}()

	listening := true
	// for listening {
	// 	select {
	// 	case <-client.EventCh():
	// 		fmt.Printf("val: %v\n", client.EventState())
	// 	case <-client.DriverCh():
	// 		fmt.Printf("val: %v\n", client.DriversState())
	// 	case err := <-client.DoneCh():
	// 		// tp.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	// 		if err != nil {
	// 			fmt.Println(err)
	// 		}
	// 		listening = false
	// 	}
	// }
	for listening {
		select {
		case <-client.EventCh():
			tp.Send(summary.RaceWeekendEventMsg{Data: client.EventState()})
		case <-client.DriverCh():
			tp.Send(summary.DriversMsg{Data: client.DriversState()})
		case <-client.DoneCh():
			tp.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		}
	}

	err = <-done
	if err != nil {
		logger.Error("TUI exited with error", "err", err)
	}
}
