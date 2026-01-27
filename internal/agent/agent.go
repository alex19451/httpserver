package agent

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/alex19451/httpserver/internal/config"
)

type Agent struct {
	cfg *config.AgentConfig
}

func New(cfg *config.AgentConfig) *Agent {
	return &Agent{cfg: cfg}
}

func (a *Agent) Run() {
	pollInterval := time.Duration(a.cfg.PollInterval) * time.Second
	reportInterval := time.Duration(a.cfg.ReportInterval) * time.Second

	fmt.Printf("Agent started\n")
	fmt.Printf("Server: %s\n", a.cfg.Address)
	fmt.Printf("Poll: %v, Report: %v\n", pollInterval, reportInterval)

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
			a.sendAll(count, mem)
		}
	}
}

func (a *Agent) sendAll(pollCount int, mem runtime.MemStats) {
	a.send("counter", "PollCount", fmt.Sprint(pollCount))
	a.send("gauge", "RandomValue", fmt.Sprint(rand.Float64()))

	a.send("gauge", "Alloc", fmt.Sprint(mem.Alloc))
	a.send("gauge", "BuckHashSys", fmt.Sprint(mem.BuckHashSys))
	a.send("gauge", "Frees", fmt.Sprint(mem.Frees))
	a.send("gauge", "GCCPUFraction", fmt.Sprint(mem.GCCPUFraction))
	a.send("gauge", "GCSys", fmt.Sprint(mem.GCSys))
	a.send("gauge", "HeapAlloc", fmt.Sprint(mem.HeapAlloc))
	a.send("gauge", "HeapIdle", fmt.Sprint(mem.HeapIdle))
	a.send("gauge", "HeapInuse", fmt.Sprint(mem.HeapInuse))
	a.send("gauge", "HeapObjects", fmt.Sprint(mem.HeapObjects))
	a.send("gauge", "HeapReleased", fmt.Sprint(mem.HeapReleased))
	a.send("gauge", "HeapSys", fmt.Sprint(mem.HeapSys))
	a.send("gauge", "LastGC", fmt.Sprint(mem.LastGC))
	a.send("gauge", "Lookups", fmt.Sprint(mem.Lookups))
	a.send("gauge", "MCacheInuse", fmt.Sprint(mem.MCacheInuse))
	a.send("gauge", "MCacheSys", fmt.Sprint(mem.MCacheSys))
	a.send("gauge", "MSpanInuse", fmt.Sprint(mem.MSpanInuse))
	a.send("gauge", "MSpanSys", fmt.Sprint(mem.MSpanSys))
	a.send("gauge", "Mallocs", fmt.Sprint(mem.Mallocs))
	a.send("gauge", "NextGC", fmt.Sprint(mem.NextGC))
	a.send("gauge", "NumForcedGC", fmt.Sprint(mem.NumForcedGC))
	a.send("gauge", "NumGC", fmt.Sprint(mem.NumGC))
	a.send("gauge", "OtherSys", fmt.Sprint(mem.OtherSys))
	a.send("gauge", "PauseTotalNs", fmt.Sprint(mem.PauseTotalNs))
	a.send("gauge", "StackInuse", fmt.Sprint(mem.StackInuse))
	a.send("gauge", "StackSys", fmt.Sprint(mem.StackSys))
	a.send("gauge", "Sys", fmt.Sprint(mem.Sys))
	a.send("gauge", "TotalAlloc", fmt.Sprint(mem.TotalAlloc))
}

func (a *Agent) send(mtype, name, value string) {
	url := fmt.Sprintf("http://%s/update/%s/%s/%s", a.cfg.Address, mtype, name, value)

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending %s/%s: %v\n", mtype, name, err)
		return
	}
	resp.Body.Close()
}
