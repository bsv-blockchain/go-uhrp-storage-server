package logger_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/logger"
)

func TestConfigureLevels(t *testing.T) {
	previous := slog.Default()

	t.Cleanup(func() { slog.SetDefault(previous) })

	tests := []struct {
		level   string
		enabled slog.Level
		hidden  slog.Level
	}{
		{level: "debug", enabled: slog.LevelDebug, hidden: slog.LevelDebug - 1},
		{level: "DEBUG", enabled: slog.LevelDebug, hidden: slog.LevelDebug - 1},
		{level: "info", enabled: slog.LevelInfo, hidden: slog.LevelDebug},
		{level: "warn", enabled: slog.LevelWarn, hidden: slog.LevelInfo},
		{level: "warning", enabled: slog.LevelWarn, hidden: slog.LevelInfo},
		{level: "error", enabled: slog.LevelError, hidden: slog.LevelWarn},
		{level: "unknown", enabled: slog.LevelInfo, hidden: slog.LevelDebug},
	}

	for _, tt := range tests {
		for _, format := range []string{"json", "text"} {
			t.Run(tt.level+"/"+format, func(t *testing.T) {
				log := logger.Configure(tt.level, format)
				ctx := t.Context()

				assert.True(t, log.Enabled(ctx, tt.enabled))
				assert.False(t, log.Enabled(ctx, tt.hidden))
				assert.Same(t, log, slog.Default(), "Configure installs the logger as the default")
			})
		}
	}
}
