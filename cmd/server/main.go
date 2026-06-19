package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "Go-Stream-Processor"})
}

var (
	processed int64
	pmu       sync.Mutex
)

func processEventHandler(w http.ResponseWriter, r *http.Request) {
	pmu.Lock()
	processed++
	pmu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	pmu.Lock()
	count := processed
	pmu.Unlock()
	json.NewEncoder(w).Encode(map[string]int64{"processed": count})
}

func init() {
	http.HandleFunc("/event", processEventHandler)
	http.HandleFunc("/stats", statsHandler)
}


func main() {
	http.HandleFunc("/health", healthHandler)
	fmt.Println("Starting Go-Stream-Processor on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
