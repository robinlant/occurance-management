package logger

import (
	"log/slog"
	"os"
)

// Init configures the default slog logger.
// In production (GIN_MODE=release) it emits JSON; otherwise human-readable text.
func Init(production bool) {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	var h slog.Handler
	if production {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}
