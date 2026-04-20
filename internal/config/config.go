package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type ServerConfig struct {
	Address         string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
	LogLevel        string
	DatabaseDSN     string
	Key             string
}

type AgentConfig struct {
	Address        string
	PollInterval   int
	ReportInterval int
	LogLevel       string
	Key            string
}

func ParseServerConfig() *ServerConfig {
	var addressFlag string
	var storeIntervalFlag int
	var fileStoragePathFlag string
	var restoreFlag bool
	var logLevelFlag string
	var databaseDSNFlag string
	var keyFlag string

	flag.StringVar(&addressFlag, "a", "localhost:8080", "HTTP server endpoint address")
	flag.IntVar(&storeIntervalFlag, "i", 300, "store interval in seconds")
	flag.StringVar(&fileStoragePathFlag, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&restoreFlag, "r", true, "restore from file on startup")
	flag.StringVar(&logLevelFlag, "l", "info", "log level (debug, info, warn, error)")
	flag.StringVar(&databaseDSNFlag, "d", "", "database DSN (postgres://user:password@host:port/dbname)")
	flag.StringVar(&keyFlag, "k", "", "key for SHA256 signature")

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
	storeInterval := getIntConfigValue("STORE_INTERVAL", storeIntervalFlag, 300)
	fileStoragePath := getConfigValue("FILE_STORAGE_PATH", fileStoragePathFlag, "/tmp/metrics-db.json")
	restore := getBoolConfigValue("RESTORE", restoreFlag, true)
	logLevel := getConfigValue("LOG_LEVEL", logLevelFlag, "info")
	databaseDSN := getConfigValue("DATABASE_DSN", databaseDSNFlag, "")
	key := getConfigValue("KEY", keyFlag, "")

	return &ServerConfig{
		Address:         address,
		StoreInterval:   storeInterval,
		FileStoragePath: fileStoragePath,
		Restore:         restore,
		LogLevel:        logLevel,
		DatabaseDSN:     databaseDSN,
		Key:             key,
	}
}

func ParseAgentConfig() *AgentConfig {
	var addressFlag string
	var pollIntervalFlag int
	var reportIntervalFlag int
	var logLevelFlag string
	var keyFlag string

	flag.StringVar(&addressFlag, "a", "localhost:8080", "HTTP server endpoint address")
	flag.IntVar(&pollIntervalFlag, "p", 2, "metrics poll interval (seconds)")
	flag.IntVar(&reportIntervalFlag, "r", 10, "metrics report interval (seconds)")
	flag.StringVar(&logLevelFlag, "l", "info", "log level (debug, info, warn, error)")
	flag.StringVar(&keyFlag, "k", "", "key for SHA256 signature")

	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	address := getConfigValue("ADDRESS", addressFlag, "localhost:8080")
	pollInterval := getIntConfigValue("POLL_INTERVAL", pollIntervalFlag, 2)
	reportInterval := getIntConfigValue("REPORT_INTERVAL", reportIntervalFlag, 10)
	logLevel := getConfigValue("LOG_LEVEL", logLevelFlag, "info")
	key := getConfigValue("KEY", keyFlag, "")

	return &AgentConfig{
		Address:        address,
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
		LogLevel:       logLevel,
		Key:            key,
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

func getBoolConfigValue(envVar string, flagValue, defaultValue bool) bool {
	if envValue := os.Getenv(envVar); envValue != "" {
		if val, err := strconv.ParseBool(envValue); err == nil {
			return val
		}
	}
	if flagValue != defaultValue {
		return flagValue
	}
	return defaultValue
}
