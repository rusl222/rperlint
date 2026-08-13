package logger

import (
	"log/slog"
	"os"
)

func Logger(debug bool) *slog.Logger {
	var logger *slog.Logger
	if debug {
		// Активация debug сообщений
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		logger = slog.New(handler).With("process", "algo")
	} else {
		logger = slog.Default().With("process", "algo")
	}
	return logger
}
