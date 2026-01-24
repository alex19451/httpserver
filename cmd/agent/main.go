package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

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
	url := fmt.Sprintf("http://localhost:8080/update/%s/%s/%s", mtype, name, value)

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Content-Type", "text/plain")

	http.DefaultClient.Do(req)
}

func main() {
	fmt.Println("Agent started")

	count := 0

	for {
		time.Sleep(2 * time.Second)
		count++

		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		if count%5 == 0 {
			fmt.Println("Sending metrics")
			sendAll(count, mem)
		}
	}
}
