package main

import (
	"io"
	"log"
	"log/slog"

	summary "github.com/bcdxn/f1cli/internal/tui/livetiming"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	done := make(chan int)
	program := tea.NewProgram(summary.New(logger, done), tea.WithAltScreen())

	go func() {
		_, err := program.Run()
		if err != nil {
			log.Fatal("Error starting TUI:", err.Error())
		}
	}()

	<-done
}
