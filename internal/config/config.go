package config

import (
	"flag"
	"fmt"
	"os"
)

type ServerConfig struct {
	Address string
}

type AgentConfig struct {
	Address        string
	PollInterval   int
	ReportInterval int
}

func ParseServerConfig() *ServerConfig {
	var address string
	flag.StringVar(&address, "a", "localhost:8080", "HTTP server endpoint address")
	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		os.Exit(1)
	}

	return &ServerConfig{Address: address}
}

func ParseAgentConfig() *AgentConfig {
	var address string
	var pollInterval int
	var reportInterval int

	flag.StringVar(&address, "a", "localhost:8080", "HTTP server endpoint address")
	flag.IntVar(&pollInterval, "p", 2, "metrics poll interval from runtime package (seconds)")
	flag.IntVar(&reportInterval, "r", 10, "metrics report interval to server (seconds)")
	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		os.Exit(1)
	}

	return &AgentConfig{
		Address:        address,
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
	}
}
