package loadtest

import (
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const requestTimeout = 30 * time.Second

// Config describes a load test run.
type Config struct {
	URL         string
	Requests    int
	Concurrency int
}

// workerResult is the per-worker tally, merged after all workers finish so the
// hot path needs no locking.
type workerResult struct {
	status map[int]int
	errors int
}

// Run performs exactly cfg.Requests GET requests against cfg.URL using
// cfg.Concurrency workers. If ctx is cancelled, workers stop early and the
// report reflects only the requests that were actually performed.
func Run(ctx context.Context, cfg Config) Report {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = cfg.Concurrency
	transport.MaxIdleConns = cfg.Concurrency
	client := &http.Client{Transport: transport, Timeout: requestTimeout}
	defer transport.CloseIdleConnections()

	var next atomic.Int64
	results := make([]workerResult, cfg.Concurrency)

	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func(res *workerResult) {
			defer wg.Done()
			res.status = make(map[int]int)
			for ctx.Err() == nil && next.Add(1) <= int64(cfg.Requests) {
				code, err := doRequest(ctx, client, cfg.URL)
				if err != nil {
					res.errors++
					continue
				}
				res.status[code]++
			}
		}(&results[i])
	}
	wg.Wait()

	report := Report{Duration: time.Since(start), Status: make(map[int]int)}
	for _, res := range results {
		report.Errors += res.errors
		report.Total += res.errors
		for code, n := range res.status {
			report.Status[code] += n
			report.Total += n
		}
	}
	return report
}

func doRequest(ctx context.Context, client *http.Client, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	// Drain the body so the connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}
