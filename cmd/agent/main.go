package main

import (
	"github.com/alex19451/httpserver/internal/agent"
	"github.com/alex19451/httpserver/internal/config"
)

func main() {
	cfg := config.ParseAgentConfig()
	ag := agent.New(cfg)
	ag.Run()
}
