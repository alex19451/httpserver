package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/server"
	"github.com/alex19451/httpserver/internal/storage"
	"github.com/rs/zerolog"
)

func main() {
	cfg := config.ParseServerConfig()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("component", "server").
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

	var db *storage.Storage
	if cfg.FileStoragePath != "" {
		db = storage.NewWithFile(cfg.FileStoragePath)
	} else {
		db = storage.New()
	}

	srv := server.New(cfg, db, logger)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		if err := srv.Run(); err != nil {
			logger.Error().Err(err).Msg("server error")
			os.Exit(1)
		}
	}()

	<-sigChan
	logger.Info().Msg("shutting down server...")

	if err := db.SaveToFile(); err != nil {
		logger.Error().Err(err).Msg("error saving data on shutdown")
	} else {
		logger.Info().Msg("data saved successfully")
	}

	os.Exit(0)
}
