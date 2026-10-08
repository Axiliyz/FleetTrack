package middleware

import (
	"fleettrack/internal/requestid"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

func TestRequestID_UsesIncomingHeader(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestid.FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestid.Header, "abc-123")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if seen != "abc-123" || rec.Header().Get(requestid.Header) != "abc-123" {
		t.Fatalf("context id %q, response header %q", seen, rec.Header().Get(requestid.Header))
	}
}

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	rec := httptest.NewRecorder()
	RequestID(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get(requestid.Header) == "" {
		t.Fatal("no request id generated")
	}
}

func TestMetrics_UnmatchedRouteAndImplicitStatus(t *testing.T) {
	r := chi.NewRouter()
	r.Use(Metrics)
	r.Get("/silent", func(http.ResponseWriter, *http.Request) {})

	for _, path := range []string{"/silent", "/no-such-route"} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	want := map[string]string{"/silent": "200", "unmatched": "404"}
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range families {
		if f.GetName() != "fleettrack_http_requests_total" {
			continue
		}
		for _, m := range f.GetMetric() {
			labels := map[string]string{}
			for _, l := range m.GetLabel() {
				labels[l.GetName()] = l.GetValue()
			}
			got[labels["path"]] = labels["status"]
		}
	}
	for path, status := range want {
		if got[path] != status {
			t.Errorf("path %q: status label %q, want %q (all: %v)", path, got[path], status, got)
		}
	}
}
