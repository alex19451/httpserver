package main

import (
	"fmt"
	"os"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/server"
	"github.com/alex19451/httpserver/internal/storage"
)

func main() {
	cfg := config.ParseServerConfig()
	db := storage.New()
	srv := server.New(cfg, db)

	if err := srv.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
