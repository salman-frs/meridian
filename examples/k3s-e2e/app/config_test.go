package e2eapp

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("MERIDIAN_ROLE", "storefront")
	t.Setenv("MERIDIAN_RUN_ID", "run-123")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.ServiceName != "storefront" {
		t.Fatalf("ServiceName = %q, want storefront", cfg.ServiceName)
	}
	if cfg.OTLPEndpoint != "otel-gateway.observability.svc.cluster.local:4317" {
		t.Fatalf("OTLPEndpoint = %q", cfg.OTLPEndpoint)
	}
	if cfg.RequestCount != 3 {
		t.Fatalf("RequestCount = %d, want 3", cfg.RequestCount)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	t.Setenv("MERIDIAN_ROLE", "inventory")
	t.Setenv("MERIDIAN_SERVICE_NAME", "inventory-api")
	t.Setenv("MERIDIAN_RUN_ID", "run-456")
	t.Setenv("MERIDIAN_DISABLE_TRACES", "true")
	t.Setenv("MERIDIAN_HEARTBEAT_INTERVAL", "2s")
	t.Setenv("MERIDIAN_EXPECT_CHECKOUT_STATUS", "502")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.DisableTraces {
		t.Fatal("DisableTraces = false, want true")
	}
	if cfg.ServiceName != "inventory-api" {
		t.Fatalf("ServiceName = %q, want inventory-api", cfg.ServiceName)
	}
	if cfg.HeartbeatInterval != 2*time.Second {
		t.Fatalf("HeartbeatInterval = %s, want 2s", cfg.HeartbeatInterval)
	}
	if cfg.ExpectedCheckoutStatus != 502 {
		t.Fatalf("ExpectedCheckoutStatus = %d, want 502", cfg.ExpectedCheckoutStatus)
	}
}

func TestEnvBoolFallbackOnInvalid(t *testing.T) {
	t.Setenv("MERIDIAN_DISABLE_METRICS", "not-a-bool")
	if got := envBool("MERIDIAN_DISABLE_METRICS", true); !got {
		t.Fatal("envBool() = false, want fallback true")
	}
}

func TestEnvOrDefaultUsesFallback(t *testing.T) {
	if err := os.Unsetenv("MERIDIAN_FAKE"); err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}
	if got := envOrDefault("MERIDIAN_FAKE", "value"); got != "value" {
		t.Fatalf("envOrDefault() = %q, want value", got)
	}
}
