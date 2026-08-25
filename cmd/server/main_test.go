package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestHealth(t *testing.T) {
	p := newProcessor()
	rr := httptest.NewRecorder()
	routes(p).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestProcessAndStats(t *testing.T) {
	p := newProcessor()
	handler := routes(p)
	for _, payload := range []string{`{"key":"orders","value":2.5}`, `{"key":"orders","value":1.5}`} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(payload)))
		if rr.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
		}
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/stats", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected stats 200, got %d", rr.Code)
	}
	var body struct {
		TotalEvents uint64               `json:"total_events"`
		UniqueKeys  int                  `json:"unique_keys"`
		Aggregates  map[string]aggregate `json:"aggregates"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.TotalEvents != 2 || body.UniqueKeys != 1 || body.Aggregates["orders"].Count != 2 || body.Aggregates["orders"].Sum != 4.0 {
		t.Fatalf("unexpected stats: %+v", body)
	}
}

func TestInvalidEventsAreRejected(t *testing.T) {
	p := newProcessor()
	handler := routes(p)
	for _, payload := range []string{
		`{"key":"bad key","value":1}`,
		`{"key":"orders","value":1000000000001}`,
		`{"key":"orders","value":1,"unknown":true}`,
		`{"key":"orders","value":1}{}`,
	} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(payload)))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d for %s", rr.Code, payload)
		}
	}
}

func TestConcurrentProcessing(t *testing.T) {
	p := newProcessor()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.process(event{Key: "concurrent", Value: 1}); err != nil {
				t.Errorf("process: %v", err)
			}
		}()
	}
	wg.Wait()
	stats := p.snapshot()
	if stats["total_events"].(uint64) != 100 {
		t.Fatalf("expected 100 events, got %v", stats["total_events"])
	}
}
