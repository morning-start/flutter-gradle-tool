package mirror

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTestSourcesSortsBySpeed(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(40 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slow.Close()

	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fast.Close()

	results := TestSources(context.Background(), []Source{
		{Name: "slow", GradleURL: slow.URL},
		{Name: "fast", GradleURL: fast.URL},
	}, 200*time.Millisecond, 2)

	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2", len(results))
	}
	if results[0].Source.Name != "fast" {
		t.Fatalf("first result = %q, want fast", results[0].Source.Name)
	}
	if !results[0].OK {
		t.Fatalf("fast result not ok: %+v", results[0])
	}
}

func TestTestSourcesPlacesFailuresLast(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	results := TestSources(context.Background(), []Source{
		{Name: "fail", GradleURL: failServer.URL},
		{Name: "ok", GradleURL: okServer.URL},
	}, 200*time.Millisecond, 2)

	if results[0].Source.Name != "ok" {
		t.Fatalf("first result = %q, want ok", results[0].Source.Name)
	}
	if results[1].Source.Name != "fail" {
		t.Fatalf("second result = %q, want fail", results[1].Source.Name)
	}
	if results[1].OK {
		t.Fatalf("failure result marked ok: %+v", results[1])
	}
}

func TestProbeResultRecordsError(t *testing.T) {
	results := TestSources(context.Background(), []Source{
		{Name: "missing", GradleURL: "http://127.0.0.1:1"},
	}, 10*time.Millisecond, 1)

	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].OK {
		t.Fatalf("expected failure for missing endpoint")
	}
	if !strings.Contains(results[0].Status, "error") {
		t.Fatalf("status = %q, want error", results[0].Status)
	}
}
