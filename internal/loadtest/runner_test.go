package loadtest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunExactRequestCount(t *testing.T) {
	cases := []struct{ requests, concurrency int }{
		{1, 1}, {10, 3}, {100, 10}, {7, 7}, {50, 1},
	}
	for _, c := range cases {
		var hits atomic.Int64
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits.Add(1)
		}))

		rep := Run(context.Background(), Config{URL: srv.URL, Requests: c.requests, Concurrency: c.concurrency})
		srv.Close()

		if int(hits.Load()) != c.requests || rep.Total != c.requests {
			t.Errorf("requests=%d concurrency=%d: server hits=%d, report total=%d", c.requests, c.concurrency, hits.Load(), rep.Total)
		}
		if rep.Status[200] != c.requests {
			t.Errorf("expected %d x 200, got %d", c.requests, rep.Status[200])
		}
	}
}

func TestRunStatusDistribution(t *testing.T) {
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch n.Add(1) % 4 {
		case 0:
			w.WriteHeader(http.StatusNotFound)
		case 1:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	rep := Run(context.Background(), Config{URL: srv.URL, Requests: 40, Concurrency: 5})

	if rep.Status[200] != 20 || rep.Status[404] != 10 || rep.Status[500] != 10 {
		t.Errorf("unexpected distribution: %v", rep.Status)
	}
	if rep.Errors != 0 || rep.Total != 40 {
		t.Errorf("errors=%d total=%d", rep.Errors, rep.Total)
	}
}

func TestRunCountsConnectionErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing is listening anymore

	rep := Run(context.Background(), Config{URL: url, Requests: 5, Concurrency: 2})

	if rep.Errors != 5 || rep.Total != 5 || len(rep.Status) != 0 {
		t.Errorf("expected 5 errors, got %+v", rep)
	}
}

func TestRunRespectsConcurrencyLimit(t *testing.T) {
	var inFlight, peak atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		cur := inFlight.Add(1)
		for {
			p := peak.Load()
			if cur <= p || peak.CompareAndSwap(p, cur) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		inFlight.Add(-1)
	}))
	defer srv.Close()

	Run(context.Background(), Config{URL: srv.URL, Requests: 40, Concurrency: 4})

	if peak.Load() > 4 {
		t.Errorf("peak concurrency %d exceeded limit 4", peak.Load())
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(5 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	rep := Run(ctx, Config{URL: srv.URL, Requests: 100000, Concurrency: 2})

	if rep.Total >= 100000 {
		t.Errorf("run was not interrupted, total=%d", rep.Total)
	}
}

func TestReportPrint(t *testing.T) {
	rep := Report{
		Duration: 2 * time.Second,
		Total:    10,
		Status:   map[int]int{200: 6, 500: 1, 404: 2},
		Errors:   1,
	}
	var buf bytes.Buffer
	rep.Print(&buf)
	out := buf.String()

	for _, want := range []string{"Total de requests:         10", "status 200:   6", "404 Not Found", "500 Internal Server Error", "Falhas de conexão/timeout: 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "404") > strings.Index(out, "500") {
		t.Errorf("status codes not sorted:\n%s", out)
	}
}
