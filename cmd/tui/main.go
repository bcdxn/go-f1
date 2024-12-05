package main

import (
	"io"
	"log"
	"log/slog"

	summary "github.com/bcdxn/f1cli/internal/tui"
)

func main() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	done := make(chan error)

	go func() {
		err := summary.RunTUI(logger, done)
		if err != nil {
			log.Fatal("Error starting TUI:", err.Error())
		}
	}()

	err := <-done
	if err != nil {
		logger.Error("TUI exited with error", "err", err)
	}
}
