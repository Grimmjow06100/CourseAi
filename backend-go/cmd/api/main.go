package main

import (
	"log/slog"
	"os"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/errtrace"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		traced := errtrace.Capture(err)
		logger.Error("application stopped", "error", err, "stack", errtrace.Stack(traced))
		os.Exit(1)
	}
}
