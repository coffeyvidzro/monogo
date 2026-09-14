package metrics

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

func Handler(registry *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		counters, gauges, err := registry.Snapshot(ctx)
		if err != nil {
			http.Error(w, "telecom metrics unavailable", http.StatusServiceUnavailable)
			return
		}

		_, _ = fmt.Fprintf(w,
			"# HELP leamout_http_requests_total Total number of HTTP requests recorded.\n"+
				"# TYPE leamout_http_requests_total counter\n"+
				"leamout_http_requests_total %d\n"+
				"# HELP leamout_http_errors_total Total number of HTTP errors recorded.\n"+
				"# TYPE leamout_http_errors_total counter\n"+
				"leamout_http_errors_total %d\n"+
				"# HELP leamout_process_uptime_seconds Process uptime in seconds.\n"+
				"# TYPE leamout_process_uptime_seconds gauge\n"+
				"leamout_process_uptime_seconds %f\n"+
				"# HELP leamout_process_goroutines Current number of goroutines.\n"+
				"# TYPE leamout_process_goroutines gauge\n"+
				"leamout_process_goroutines %d\n",
			registry.Requests(),
			registry.Errors(),
			time.Since(startTime).Seconds(),
			runtime.NumGoroutine(),
		)
		writeTelecom(w, counters, "counter")
		writeTelecom(w, gauges, "gauge")
	})
}

func writeTelecom(w http.ResponseWriter, values map[string]string, metricType string) {
	fields := make([]string, 0, len(values))
	for field := range values {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	declared := make(map[string]struct{})
	for _, field := range fields {
		name, labels, err := ParseSeries(field)
		if err != nil {
			continue
		}
		value := values[field]
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			continue
		}
		if _, ok := declared[name]; !ok {
			_, _ = fmt.Fprintf(w, "# TYPE leamout_%s %s\n", name, metricType)
			declared[name] = struct{}{}
		}
		_, _ = fmt.Fprintf(w, "leamout_%s%s %s\n", name, prometheusLabels(labels), value)
	}
}

func prometheusLabels(labels [4]string) string {
	names := [4]string{"carrier_connection_id", "trunk_id", "endpoint_id", "result"}
	items := make([]string, 0, 4)
	for index, value := range labels {
		if value != "" {
			items = append(items, names[index]+`="`+strings.ReplaceAll(value, `"`, `\"`)+`"`)
		}
	}
	if len(items) == 0 {
		return ""
	}
	return "{" + strings.Join(items, ",") + "}"
}

var startTime = time.Now()
