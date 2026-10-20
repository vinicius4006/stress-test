package loadtest

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// Report summarises a load test run. Total includes requests that failed
// before getting an HTTP response (counted in Errors).
type Report struct {
	Duration time.Duration
	Total    int
	Status   map[int]int
	Errors   int
}

// Print writes a human-readable report to w.
func (r Report) Print(w io.Writer) {
	fmt.Fprintln(w, "========== Relatório do teste de carga ==========")
	fmt.Fprintf(w, "Tempo total:               %s\n", r.Duration.Round(time.Millisecond))
	fmt.Fprintf(w, "Total de requests:         %d\n", r.Total)
	if secs := r.Duration.Seconds(); secs > 0 {
		fmt.Fprintf(w, "Requests por segundo:      %.2f\n", float64(r.Total)/secs)
	}
	fmt.Fprintf(w, "Requests com status 200:   %d\n", r.Status[http.StatusOK])

	codes := make([]int, 0, len(r.Status))
	for code := range r.Status {
		if code != http.StatusOK {
			codes = append(codes, code)
		}
	}
	sort.Ints(codes)

	if len(codes) > 0 {
		fmt.Fprintln(w, "Outros status HTTP:")
		for _, code := range codes {
			fmt.Fprintf(w, "  %d %-24s %d\n", code, http.StatusText(code), r.Status[code])
		}
	}
	if r.Errors > 0 {
		fmt.Fprintf(w, "Falhas de conexão/timeout: %d\n", r.Errors)
	}
}
