package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address        string `env:"ADDRESS" envDefault:"localhost:8080"`
	PollInterval   int    `env:"POLL_INTERVAL" envDefault:"2"`
	ReportInterval int    `env:"REPORT_INTERVAL" envDefault:"10"`
}

var cfg Config

func sendAll(pollCount int, mem runtime.MemStats) {
	send("counter", "PollCount", fmt.Sprint(pollCount))
	send("gauge", "RandomValue", fmt.Sprint(rand.Float64()))

	send("gauge", "Alloc", fmt.Sprint(mem.Alloc))
	send("gauge", "BuckHashSys", fmt.Sprint(mem.BuckHashSys))
	send("gauge", "Frees", fmt.Sprint(mem.Frees))
	send("gauge", "GCCPUFraction", fmt.Sprint(mem.GCCPUFraction))
	send("gauge", "GCSys", fmt.Sprint(mem.GCSys))
	send("gauge", "HeapAlloc", fmt.Sprint(mem.HeapAlloc))
	send("gauge", "HeapIdle", fmt.Sprint(mem.HeapIdle))
	send("gauge", "HeapInuse", fmt.Sprint(mem.HeapInuse))
	send("gauge", "HeapObjects", fmt.Sprint(mem.HeapObjects))
	send("gauge", "HeapReleased", fmt.Sprint(mem.HeapReleased))
	send("gauge", "HeapSys", fmt.Sprint(mem.HeapSys))
	send("gauge", "LastGC", fmt.Sprint(mem.LastGC))
	send("gauge", "Lookups", fmt.Sprint(mem.Lookups))
	send("gauge", "MCacheInuse", fmt.Sprint(mem.MCacheInuse))
	send("gauge", "MCacheSys", fmt.Sprint(mem.MCacheSys))
	send("gauge", "MSpanInuse", fmt.Sprint(mem.MSpanInuse))
	send("gauge", "MSpanSys", fmt.Sprint(mem.MSpanSys))
	send("gauge", "Mallocs", fmt.Sprint(mem.Mallocs))
	send("gauge", "NextGC", fmt.Sprint(mem.NextGC))
	send("gauge", "NumForcedGC", fmt.Sprint(mem.NumForcedGC))
	send("gauge", "NumGC", fmt.Sprint(mem.NumGC))
	send("gauge", "OtherSys", fmt.Sprint(mem.OtherSys))
	send("gauge", "PauseTotalNs", fmt.Sprint(mem.PauseTotalNs))
	send("gauge", "StackInuse", fmt.Sprint(mem.StackInuse))
	send("gauge", "StackSys", fmt.Sprint(mem.StackSys))
	send("gauge", "Sys", fmt.Sprint(mem.Sys))
	send("gauge", "TotalAlloc", fmt.Sprint(mem.TotalAlloc))
}

func send(mtype, name, value string) {
	url := fmt.Sprintf("http://%s/update/%s/%s/%s", cfg.Address, mtype, name, value)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		fmt.Printf("Error creating request for %s/%s: %v\n", mtype, name, err)
		return
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending metric %s/%s: %v\n", mtype, name, err)
		return
	}
	resp.Body.Close()
}

func main() {
	var addressFlag string
	var pollIntervalFlag int
	var reportIntervalFlag int

	flag.StringVar(&addressFlag, "a", "localhost:8080", "HTTP server endpoint address")
	flag.IntVar(&pollIntervalFlag, "p", 2, "metrics poll interval from runtime package (seconds)")
	flag.IntVar(&reportIntervalFlag, "r", 10, "metrics report interval to server (seconds)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	if err := env.Parse(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing environment variables: %v\n", err)
		os.Exit(1)
	}

	if addressFlag != "localhost:8080" {
		cfg.Address = addressFlag
	}
	if pollIntervalFlag != 2 {
		cfg.PollInterval = pollIntervalFlag
	}
	if reportIntervalFlag != 10 {
		cfg.ReportInterval = reportIntervalFlag
	}

	pollInterval := time.Duration(cfg.PollInterval) * time.Second
	reportInterval := time.Duration(cfg.ReportInterval) * time.Second

	fmt.Printf("Agent started\n")
	fmt.Printf("Server address: %s\n", cfg.Address)
	fmt.Printf("Poll interval: %v\n", pollInterval)
	fmt.Printf("Report interval: %v\n", reportInterval)

	count := 0
	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	var mem runtime.MemStats

	for {
		select {
		case <-pollTicker.C:
			count++
			runtime.ReadMemStats(&mem)

		case <-reportTicker.C:
			fmt.Println("Sending metrics")
			sendAll(count, mem)
		}
	}
}
