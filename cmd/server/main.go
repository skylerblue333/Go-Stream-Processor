package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"
)

const maxKeys = 256

var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

type event struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
}

type aggregate struct {
	Count uint64  `json:"count"`
	Sum   float64 `json:"sum"`
}

type processor struct {
	mu     sync.RWMutex
	byKey  map[string]aggregate
	total  uint64
	reject uint64
}

func newProcessor() *processor {
	return &processor{byKey: make(map[string]aggregate)}
}

func (p *processor) process(input event) error {
	if !keyPattern.MatchString(input.Key) {
		p.recordReject()
		return errors.New("key must match [A-Za-z0-9][A-Za-z0-9._-]{0,63}")
	}
	if math.IsNaN(input.Value) || math.IsInf(input.Value, 0) || math.Abs(input.Value) > 1e12 {
		p.recordReject()
		return errors.New("value must be finite and within +/-1e12")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	current, exists := p.byKey[input.Key]
	if !exists && len(p.byKey) >= maxKeys {
		p.reject++
		return errors.New("key cardinality limit reached")
	}
	if current.Count == ^uint64(0) || p.total == ^uint64(0) {
		p.reject++
		return errors.New("event counter overflow")
	}
	nextSum := current.Sum + input.Value
	if math.IsNaN(nextSum) || math.IsInf(nextSum, 0) {
		p.reject++
		return errors.New("aggregate sum overflow")
	}
	current.Count++
	current.Sum = nextSum
	p.byKey[input.Key] = current
	p.total++
	return nil
}

func (p *processor) recordReject() {
	p.mu.Lock()
	p.reject++
	p.mu.Unlock()
}

func (p *processor) snapshot() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()
	copyByKey := make(map[string]aggregate, len(p.byKey))
	for key, value := range p.byKey {
		copyByKey[key] = value
	}
	return map[string]any{
		"total_events":    p.total,
		"rejected_events": p.reject,
		"unique_keys":     len(p.byKey),
		"aggregates":      copyByKey,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func routes(p *processor) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "sky-stream-aggregate"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var input event
		if err := decoder.Decode(&input); err != nil {
			p.recordReject()
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event payload"})
			return
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			p.recordReject()
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only one complete JSON object is allowed"})
			return
		}
		if err := p.process(input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
	})
	mux.HandleFunc("/v1/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, p.snapshot())
	})
	return mux
}

func main() {
	p := newProcessor()
	server := &http.Server{
		Addr:              ":8080",
		Handler:           routes(p),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.Printf("Sky Stream Aggregate listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
