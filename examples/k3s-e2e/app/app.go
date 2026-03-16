package e2eapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type serviceCaller func(ctx context.Context, url string, token string, runID string) error

func Run(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	switch cfg.Role {
	case "storefront":
		return runStorefront(ctx, cfg)
	case "checkout":
		return runCheckout(ctx, cfg)
	case "inventory":
		return runInventory(ctx, cfg)
	case "trafficgen":
		return runTrafficgen(ctx, cfg)
	default:
		return fmt.Errorf("unsupported MERIDIAN_ROLE %q", cfg.Role)
	}
}

func runStorefront(ctx context.Context, cfg Config) error {
	telemetry, err := newTelemetry(ctx, cfg)
	if err != nil {
		return err
	}
	defer telemetry.shutdown(context.Background())

	client := newServiceClient(telemetry)
	return serveHTTP(ctx, cfg, newStorefrontHandler(cfg, telemetry, makeServiceCaller(client)))
}

func runCheckout(ctx context.Context, cfg Config) error {
	telemetry, err := newTelemetry(ctx, cfg)
	if err != nil {
		return err
	}
	defer telemetry.shutdown(context.Background())

	return serveHTTP(ctx, cfg, newCheckoutHandler(cfg, telemetry))
}

func runInventory(ctx context.Context, cfg Config) error {
	telemetry, err := newTelemetry(ctx, cfg)
	if err != nil {
		return err
	}
	defer telemetry.shutdown(context.Background())

	go heartbeatLoop(ctx, cfg, telemetry)
	return serveHTTP(ctx, cfg, newInventoryHandler(cfg, telemetry))
}

func newStorefrontHandler(cfg Config, telemetry *telemetry, call serviceCaller) http.Handler {
	mux := newHealthMux()
	mux.HandleFunc("/browse", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := telemetry.tracer.Start(r.Context(), "storefront.browse")
		defer span.End()

		telemetry.requests.Add(ctx, 1, commonMetricOption(cfg,
			attribute.String("route", "/browse"),
			attribute.String("method", r.Method),
		))
		logEvent(cfg, "info", "browse_request", map[string]any{"path": r.URL.Path, "method": r.Method})
		writeOK(w, "browse ok")
	})
	mux.HandleFunc("/checkout", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := telemetry.tracer.Start(r.Context(), "storefront.checkout")
		defer span.End()

		telemetry.requests.Add(ctx, 1, commonMetricOption(cfg,
			attribute.String("route", "/checkout"),
			attribute.String("method", r.Method),
		))

		runID := headerOrDefault(r, "X-Meridian-Run-ID", cfg.RunID)
		if err := call(ctx, cfg.InventoryURL+"/reserve", cfg.OutboundToken, runID); err != nil {
			handleDependencyError(cfg, span, w, "inventory", err)
			return
		}
		if err := call(ctx, cfg.CheckoutURL+"/checkout", cfg.OutboundToken, runID); err != nil {
			handleDependencyError(cfg, span, w, "checkout", err)
			return
		}

		logEvent(cfg, "info", "checkout_request", map[string]any{"path": r.URL.Path, "method": r.Method})
		writeOK(w, "checkout ok")
	})
	return mux
}

func newCheckoutHandler(cfg Config, telemetry *telemetry) http.Handler {
	mux := newHealthMux()
	mux.HandleFunc("/checkout", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := telemetry.tracer.Start(r.Context(), "checkout.process")
		defer span.End()

		if !tokenMatches(r, cfg.ExpectedToken) {
			span.SetStatus(codes.Error, "auth_error")
			span.SetAttributes(attribute.String("auth.result", "denied"))
			logEvent(cfg, "error", "auth_error", map[string]any{"path": r.URL.Path, "service": cfg.ServiceName})
			http.Error(w, "auth failed", http.StatusUnauthorized)
			return
		}

		telemetry.checkouts.Add(ctx, 1, commonMetricOption(cfg,
			attribute.String("route", "/checkout"),
			attribute.String("method", r.Method),
		))
		logEvent(cfg, "info", "checkout_processed", map[string]any{"path": r.URL.Path})
		writeOK(w, "processed")
	})
	return mux
}

func newInventoryHandler(cfg Config, telemetry *telemetry) http.Handler {
	mux := newHealthMux()
	mux.HandleFunc("/reserve", func(w http.ResponseWriter, r *http.Request) {
		_, span := telemetry.tracer.Start(r.Context(), "inventory.reserve")
		defer span.End()

		if !tokenMatches(r, cfg.ExpectedToken) {
			span.SetStatus(codes.Error, "auth_error")
			logEvent(cfg, "error", "auth_error", map[string]any{"path": r.URL.Path, "service": cfg.ServiceName})
			http.Error(w, "auth failed", http.StatusUnauthorized)
			return
		}

		logEvent(cfg, "info", "inventory_reserved", map[string]any{"path": r.URL.Path})
		writeOK(w, "reserved")
	})
	return mux
}

func newHealthMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeOK(w, "ok")
	})
	return mux
}

func newServiceClient(telemetry *telemetry) *http.Client {
	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: telemetry.client(),
	}
}

func makeServiceCaller(client *http.Client) serviceCaller {
	return func(ctx context.Context, url string, token string, runID string) error {
		return callService(ctx, client, url, token, runID)
	}
}

func handleDependencyError(cfg Config, span trace.Span, w http.ResponseWriter, dependency string, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	logEvent(cfg, "error", "checkout_dependency_error", map[string]any{"dependency": dependency, "error": err.Error()})
	http.Error(w, err.Error(), http.StatusBadGateway)
}

func writeOK(w http.ResponseWriter, body string) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func runTrafficgen(ctx context.Context, cfg Config) error {
	client := &http.Client{Timeout: 5 * time.Second}
	logEvent(cfg, "info", "trafficgen_started", map[string]any{"request_count": cfg.RequestCount})

	for i := 0; i < cfg.RequestCount; i++ {
		if err := runTrafficStep(ctx, client, cfg, cfg.StorefrontURL+"/browse", cfg.ExpectedBrowseStatus); err != nil {
			return err
		}
		if err := runTrafficStep(ctx, client, cfg, cfg.StorefrontURL+"/checkout", cfg.ExpectedCheckoutStatus); err != nil {
			return err
		}
		time.Sleep(cfg.RequestDelay)
	}

	logEvent(cfg, "info", "trafficgen_completed", map[string]any{"request_count": cfg.RequestCount})
	return nil
}

func runTrafficStep(ctx context.Context, client *http.Client, cfg Config, url string, expectedStatus int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Meridian-Run-ID", cfg.RunID)

	resp, err := client.Do(req)
	if err != nil {
		logEvent(cfg, "error", "trafficgen_request_error", map[string]any{"url": url, "error": err.Error()})
		return err
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected status for %s: got %d want %d", url, resp.StatusCode, expectedStatus)
	}

	logEvent(cfg, "info", "trafficgen_request_ok", map[string]any{"url": url, "status": resp.StatusCode})
	return nil
}

func heartbeatLoop(ctx context.Context, cfg Config, telemetry *telemetry) {
	ticker := time.NewTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			heartbeatCtx, span := telemetry.tracer.Start(ctx, "inventory.heartbeat")
			telemetry.heartbeats.Add(heartbeatCtx, 1, commonMetricOption(cfg,
				attribute.String("route", "heartbeat"),
				attribute.String("method", "tick"),
			))
			logEvent(cfg, "info", "inventory_heartbeat", map[string]any{"interval": cfg.HeartbeatInterval.String()})
			span.End()
		}
	}
}

func callService(ctx context.Context, client *http.Client, url string, token string, runID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("X-Meridian-Token", token)
	}
	req.Header.Set("X-Meridian-Run-ID", runID)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("upstream %s returned %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func serveHTTP(ctx context.Context, cfg Config, handler http.Handler) error {
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	stopCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logEvent(cfg, "info", "server_started", map[string]any{"port": cfg.Port})
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-stopCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func tokenMatches(r *http.Request, expected string) bool {
	if expected == "" {
		return true
	}
	return r.Header.Get("X-Meridian-Token") == expected
}

func headerOrDefault(r *http.Request, key string, fallback string) string {
	value := r.Header.Get(key)
	if value == "" {
		return fallback
	}
	return value
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var statusErr *statusError
	if errors.As(err, &statusErr) {
		return statusErr.code
	}
	return 1
}

type statusError struct {
	code int
	err  error
}

func (e *statusError) Error() string {
	return e.err.Error()
}

func withStatus(code int, err error) error {
	if err == nil {
		return nil
	}
	return &statusError{code: code, err: err}
}

func parseExpectedStatus(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
