package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/server"
	"github.com/alex19451/httpserver/internal/storage"
)

func main() {
	cfg := config.ParseServerConfig()

	var db *storage.Storage
	if cfg.FileStoragePath != "" {
		db = storage.NewWithFile(cfg.FileStoragePath)
	} else {
		db = storage.New()
	}

	srv := server.New(cfg, db)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		if err := srv.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-sigChan
	fmt.Println("\nShutting down server...")

	if err := db.SaveToFile(); err != nil {
		fmt.Printf("Error saving data on shutdown: %v\n", err)
	} else {
		fmt.Println("Data saved successfully")
	}

	os.Exit(0)
}
