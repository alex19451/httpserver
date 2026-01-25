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

func sendAll(pollCount int, mem runtime.MemStats, address string) {
	send("counter", "PollCount", fmt.Sprint(pollCount), address)
	send("gauge", "RandomValue", fmt.Sprint(rand.Float64()), address)

	send("gauge", "Alloc", fmt.Sprint(mem.Alloc), address)
	send("gauge", "BuckHashSys", fmt.Sprint(mem.BuckHashSys), address)
	send("gauge", "Frees", fmt.Sprint(mem.Frees), address)
	send("gauge", "GCCPUFraction", fmt.Sprint(mem.GCCPUFraction), address)
	send("gauge", "GCSys", fmt.Sprint(mem.GCSys), address)
	send("gauge", "HeapAlloc", fmt.Sprint(mem.HeapAlloc), address)
	send("gauge", "HeapIdle", fmt.Sprint(mem.HeapIdle), address)
	send("gauge", "HeapInuse", fmt.Sprint(mem.HeapInuse), address)
	send("gauge", "HeapObjects", fmt.Sprint(mem.HeapObjects), address)
	send("gauge", "HeapReleased", fmt.Sprint(mem.HeapReleased), address)
	send("gauge", "HeapSys", fmt.Sprint(mem.HeapSys), address)
	send("gauge", "LastGC", fmt.Sprint(mem.LastGC), address)
	send("gauge", "Lookups", fmt.Sprint(mem.Lookups), address)
	send("gauge", "MCacheInuse", fmt.Sprint(mem.MCacheInuse), address)
	send("gauge", "MCacheSys", fmt.Sprint(mem.MCacheSys), address)
	send("gauge", "MSpanInuse", fmt.Sprint(mem.MSpanInuse), address)
	send("gauge", "MSpanSys", fmt.Sprint(mem.MSpanSys), address)
	send("gauge", "Mallocs", fmt.Sprint(mem.Mallocs), address)
	send("gauge", "NextGC", fmt.Sprint(mem.NextGC), address)
	send("gauge", "NumForcedGC", fmt.Sprint(mem.NumForcedGC), address)
	send("gauge", "NumGC", fmt.Sprint(mem.NumGC), address)
	send("gauge", "OtherSys", fmt.Sprint(mem.OtherSys), address)
	send("gauge", "PauseTotalNs", fmt.Sprint(mem.PauseTotalNs), address)
	send("gauge", "StackInuse", fmt.Sprint(mem.StackInuse), address)
	send("gauge", "StackSys", fmt.Sprint(mem.StackSys), address)
	send("gauge", "Sys", fmt.Sprint(mem.Sys), address)
	send("gauge", "TotalAlloc", fmt.Sprint(mem.TotalAlloc), address)
}

func send(mtype, name, value, address string) {
	url := fmt.Sprintf("http://%s/update/%s/%s/%s", address, mtype, name, value)

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
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing environment variables: %v\n", err)
		os.Exit(1)
	}

	var addressFlag string
	var pollIntervalFlag int
	var reportIntervalFlag int

	flag.StringVar(&addressFlag, "a", cfg.Address, "HTTP server endpoint address")
	flag.IntVar(&pollIntervalFlag, "p", cfg.PollInterval, "metrics poll interval from runtime package (seconds)")
	flag.IntVar(&reportIntervalFlag, "r", cfg.ReportInterval, "metrics report interval to server (seconds)")

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

	finalAddress := addressFlag
	finalPollInterval := pollIntervalFlag
	finalReportInterval := reportIntervalFlag

	pollInterval := time.Duration(finalPollInterval) * time.Second
	reportInterval := time.Duration(finalReportInterval) * time.Second

	fmt.Printf("Agent started\n")
	fmt.Printf("Server address: %s\n", finalAddress)
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
			sendAll(count, mem, finalAddress)
		}
	}
}
