package e2eapp

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewStorefrontHandlerBrowse(t *testing.T) {
	cfg := testConfig("storefront")
	telemetry := newTestTelemetry(t, cfg)
	handler := newStorefrontHandler(cfg, telemetry, func(context.Context, string, string, string) error {
		t.Fatal("service caller should not be used for /browse")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/browse", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/browse status = %d, want 200", rec.Code)
	}
}

func TestNewStorefrontHandlerCheckout(t *testing.T) {
	tests := []struct {
		name          string
		callErrs      map[string]error
		wantStatus    int
		wantCallCount int
	}{
		{
			name:          "success",
			callErrs:      map[string]error{},
			wantStatus:    http.StatusOK,
			wantCallCount: 2,
		},
		{
			name: "dependency failure",
			callErrs: map[string]error{
				"http://inventory:8080/reserve": errors.New("inventory failed"),
			},
			wantStatus:    http.StatusBadGateway,
			wantCallCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig("storefront")
			telemetry := newTestTelemetry(t, cfg)
			var calls []string
			handler := newStorefrontHandler(cfg, telemetry, func(_ context.Context, url string, token string, runID string) error {
				calls = append(calls, strings.Join([]string{url, token, runID}, "|"))
				return tt.callErrs[url]
			})

			req := httptest.NewRequest(http.MethodGet, "/checkout", nil)
			req.Header.Set("X-Meridian-Run-ID", "header-run")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("/checkout status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if len(calls) != tt.wantCallCount {
				t.Fatalf("service calls = %d, want %d", len(calls), tt.wantCallCount)
			}
			if len(calls) > 0 && !strings.HasSuffix(calls[0], "|header-run") {
				t.Fatalf("first call = %q, want run id from request header", calls[0])
			}
		})
	}
}

func TestNewCheckoutHandler(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "authorized", token: "meridian-shared-secret", wantStatus: http.StatusOK},
		{name: "unauthorized", token: "wrong-token", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig("checkout")
			telemetry := newTestTelemetry(t, cfg)
			handler := newCheckoutHandler(cfg, telemetry)

			req := httptest.NewRequest(http.MethodGet, "/checkout", nil)
			req.Header.Set("X-Meridian-Token", tt.token)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestNewInventoryHandler(t *testing.T) {
	cfg := testConfig("inventory")
	telemetry := newTestTelemetry(t, cfg)
	handler := newInventoryHandler(cfg, telemetry)

	req := httptest.NewRequest(http.MethodGet, "/reserve", nil)
	req.Header.Set("X-Meridian-Token", cfg.ExpectedToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestRunTrafficStep(t *testing.T) {
	var gotRunID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRunID = r.Header.Get("X-Meridian-Run-ID")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	cfg := testConfig("trafficgen")
	cfg.RunID = "run-traffic"

	err := runTrafficStep(context.Background(), server.Client(), cfg, server.URL, http.StatusAccepted)
	if err != nil {
		t.Fatalf("runTrafficStep() error = %v", err)
	}
	if gotRunID != "run-traffic" {
		t.Fatalf("run id header = %q, want run-traffic", gotRunID)
	}
}

func TestRunTrafficStepUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := runTrafficStep(context.Background(), server.Client(), testConfig("trafficgen"), server.URL, http.StatusOK)
	if err == nil {
		t.Fatal("runTrafficStep() error = nil, want error")
	}
}

func TestHelperFunctions(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/checkout", nil)
	req.Header.Set("X-Test", "from-header")
	req.Header.Set("X-Meridian-Token", "secret")

	if got := headerOrDefault(req, "X-Test", "fallback"); got != "from-header" {
		t.Fatalf("headerOrDefault() = %q, want from-header", got)
	}
	if got := headerOrDefault(req, "X-Missing", "fallback"); got != "fallback" {
		t.Fatalf("headerOrDefault() = %q, want fallback", got)
	}
	if !tokenMatches(req, "secret") {
		t.Fatal("tokenMatches() = false, want true")
	}
	if tokenMatches(req, "wrong") {
		t.Fatal("tokenMatches() = true, want false")
	}
	if got := parseExpectedStatus("502", 200); got != 502 {
		t.Fatalf("parseExpectedStatus() = %d, want 502", got)
	}
	if got := parseExpectedStatus("bad", 200); got != 200 {
		t.Fatalf("parseExpectedStatus() = %d, want 200", got)
	}
	if got := ExitCode(withStatus(7, errors.New("boom"))); got != 7 {
		t.Fatalf("ExitCode() = %d, want 7", got)
	}
}

func newTestTelemetry(t *testing.T, cfg Config) *telemetry {
	t.Helper()

	cfg.DisableMetrics = true
	cfg.DisableTraces = true
	cfg.DisableAppLogs = true

	var buf bytes.Buffer
	previous := logOutput
	logOutput = &buf
	t.Cleanup(func() {
		logOutput = previous
	})

	telemetry, err := newTelemetry(context.Background(), cfg)
	if err != nil {
		t.Fatalf("newTelemetry() error = %v", err)
	}
	t.Cleanup(func() {
		_ = telemetry.shutdown(context.Background())
	})
	return telemetry
}

func testConfig(role string) Config {
	return Config{
		Role:                   role,
		ServiceName:            role,
		Port:                   "8080",
		RunID:                  "run-123",
		Scenario:               "happy",
		ExpectedToken:          "meridian-shared-secret",
		OutboundToken:          "meridian-shared-secret",
		StorefrontURL:          "http://storefront:8080",
		CheckoutURL:            "http://checkout:8080",
		InventoryURL:           "http://inventory:8080",
		HeartbeatInterval:      5 * time.Millisecond,
		RequestCount:           1,
		RequestDelay:           1 * time.Millisecond,
		ExpectedBrowseStatus:   http.StatusOK,
		ExpectedCheckoutStatus: http.StatusOK,
	}
}
