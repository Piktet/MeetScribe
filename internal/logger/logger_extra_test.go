package logger

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevelFromString(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  string
	}{
		{name: "debug", level: "debug", want: "DEBUG"},
		{name: "debug upper", level: "DEBUG", want: "DEBUG"},
		{name: "warn", level: "warn", want: "WARN"},
		{name: "warning", level: "warning", want: "WARN"},
		{name: "error", level: "ERROR", want: "ERROR"},
		{name: "info default", level: "info", want: "INFO"},
		{name: "unknown default", level: "nonsense", want: "INFO"},
		{name: "empty default", level: "", want: "INFO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, levelFromString(tt.level).String())
		})
	}
}

// TestInitLoggerLevels проверяет установку различных уровней логирования.
func TestInitLoggerLevels(t *testing.T) {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR", "unknown"}
	for _, lvl := range levels {
		assert.NoError(t, InitLogger(lvl))
	}
}

// TestLogFunctions проверяет, что функции логирования не паникуют.
func TestLogFunctions(t *testing.T) {
	assert.NoError(t, InitLogger("DEBUG"))

	Info("info message", "key", "value")
	Warn("warn message", "key", "value")
	Debug("debug message", "key", "value")
	Error(errors.New("some error"), "error message")
	Error(nil, "error message without error", "key", "value")

	l := With("component", "test")
	assert.NotNil(t, l)
	l.Info("message from With logger")
}
