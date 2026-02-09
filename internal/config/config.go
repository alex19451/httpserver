package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
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
	var addressFlag string
	flag.StringVar(&addressFlag, "a", "localhost:8080", "HTTP server endpoint address")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	address := getConfigValue("ADDRESS", addressFlag, "localhost:8080")
	return &ServerConfig{Address: address}
}

func ParseAgentConfig() *AgentConfig {
	var addressFlag string
	var pollIntervalFlag int
	var reportIntervalFlag int

	flag.StringVar(&addressFlag, "a", "localhost:8080", "HTTP server endpoint address")
	flag.IntVar(&pollIntervalFlag, "p", 2, "metrics poll interval (seconds)")
	flag.IntVar(&reportIntervalFlag, "r", 10, "metrics report interval (seconds)")

	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	address := getConfigValue("ADDRESS", addressFlag, "localhost:8080")
	pollInterval := getIntConfigValue("POLL_INTERVAL", pollIntervalFlag, 2)
	reportInterval := getIntConfigValue("REPORT_INTERVAL", reportIntervalFlag, 10)

	return &AgentConfig{
		Address:        address,
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
	}
}

func getConfigValue(envVar, flagValue, defaultValue string) string {
	if envValue := os.Getenv(envVar); envValue != "" {
		return envValue
	}

	if envVar == "ADDRESS" {
		if port := os.Getenv("SERVER_PORT"); port != "" {
			return "localhost:" + port
		}
	}

	if flagValue != "" && flagValue != defaultValue {
		return flagValue
	}
	return defaultValue
}

func getIntConfigValue(envVar string, flagValue, defaultValue int) int {
	if envValue := os.Getenv(envVar); envValue != "" {
		if val, err := strconv.Atoi(envValue); err == nil {
			return val
		}
	}
	if flagValue != defaultValue {
		return flagValue
	}
	return defaultValue
}
