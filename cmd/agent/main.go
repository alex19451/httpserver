package main

import (
	"os"

	"github.com/alex19451/httpserver/internal/agent"
	"github.com/alex19451/httpserver/internal/config"
	"github.com/rs/zerolog"
)

func main() {
	cfg := config.ParseAgentConfig()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("component", "agent").
		Logger()

	switch cfg.LogLevel {
	case "debug":
		logger = logger.Level(zerolog.DebugLevel)
	case "info":
		logger = logger.Level(zerolog.InfoLevel)
	case "warn":
		logger = logger.Level(zerolog.WarnLevel)
	case "error":
		logger = logger.Level(zerolog.ErrorLevel)
	default:
		logger = logger.Level(zerolog.InfoLevel)
	}

	ag := agent.New(cfg, logger)
	ag.Run()
}
