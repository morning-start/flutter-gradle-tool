package mirror

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

type ProbeResult struct {
	Source    Source
	OK        bool
	Status    string
	Duration  time.Duration
	Error     string
	TestedURL string
}

func TestSources(ctx context.Context, sources []Source, timeout time.Duration, concurrency int) []ProbeResult {
	if concurrency <= 0 {
		concurrency = 4
	}

	sem := make(chan struct{}, concurrency)
	results := make([]ProbeResult, len(sources))
	var wg sync.WaitGroup

	for i := range sources {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = probeSource(ctx, sources[idx], timeout)
		}(i)
	}

	wg.Wait()

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].OK != results[j].OK {
			return results[i].OK && !results[j].OK
		}
		if results[i].Duration == results[j].Duration {
			return results[i].Source.Name < results[j].Source.Name
		}
		return results[i].Duration < results[j].Duration
	})

	return results
}

func probeSource(ctx context.Context, source Source, timeout time.Duration) ProbeResult {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodHead, source.GradleURL, nil)
	if err != nil {
		return ProbeResult{Source: source, OK: false, Status: "error", Error: err.Error()}
	}

	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)
	if err != nil {
		return ProbeResult{Source: source, OK: false, Status: "error", Duration: duration, Error: err.Error(), TestedURL: source.GradleURL}
	}
	defer resp.Body.Close()

	ok := resp.StatusCode >= 200 && resp.StatusCode < 400
	status := fmt.Sprintf("%d", resp.StatusCode)
	if !ok {
		status = "error"
	}

	return ProbeResult{
		Source:    source,
		OK:        ok,
		Status:    status,
		Duration:  duration,
		TestedURL: source.GradleURL,
	}
}
