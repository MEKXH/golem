package commands

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/MEKXH/golem/internal/config"
)

var (
	loggerMu      sync.Mutex
	activeLogFile *os.File
)

func configureLogger(cfg *config.Config, overrideLevel string, tuiMode bool) error {
	level, err := parseLogLevel(cfg.Log.Level, overrideLevel)
	if err != nil {
		return err
	}

	writer := io.Writer(os.Stderr)
	logFilePath := strings.TrimSpace(cfg.Log.File)

	loggerMu.Lock()
	defer loggerMu.Unlock()

	if activeLogFile != nil && (logFilePath == "" || activeLogFile.Name() != logFilePath) {
		_ = activeLogFile.Close()
		activeLogFile = nil
	}

	if logFilePath != "" {
		if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
			return fmt.Errorf("create log directory: %w", err)
		}
		if activeLogFile == nil {
			f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("open log file: %w", err)
			}
			activeLogFile = f
		}
		writer = activeLogFile
	} else if tuiMode {
		writer = io.Discard
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
	return nil
}

func parseLogLevel(configLevel, override string) (slog.Level, error) {
	level := strings.TrimSpace(configLevel)
	if strings.TrimSpace(override) != "" {
		level = override
	}
	switch strings.ToLower(level) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level: %s", level)
	}
}
