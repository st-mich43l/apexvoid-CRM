package logging

import (
	"log/slog"
	"os"
	"strings"
)

func New(environment, level string) *slog.Logger {
	var handler slog.Handler
	options := &slog.HandlerOptions{Level: parseLevel(level)}
	if strings.EqualFold(environment, "production") {
		handler = slog.NewJSONHandler(os.Stdout, options)
	} else {
		handler = slog.NewTextHandler(os.Stdout, options)
	}
	return slog.New(handler).With("service", "apexvoid-crm", "environment", environment)
}

func parseLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
