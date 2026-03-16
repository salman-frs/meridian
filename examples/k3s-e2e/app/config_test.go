package e2eapp

import (
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name   string
		env    map[string]string
		assert func(*testing.T, Config)
	}{
		{
			name: "defaults",
			env: map[string]string{
				"MERIDIAN_ROLE":   "storefront",
				"MERIDIAN_RUN_ID": "run-123",
			},
			assert: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.ServiceName != "storefront" {
					t.Fatalf("ServiceName = %q, want storefront", cfg.ServiceName)
				}
				if cfg.OTLPEndpoint != "otel-gateway.observability.svc.cluster.local:4317" {
					t.Fatalf("OTLPEndpoint = %q", cfg.OTLPEndpoint)
				}
				if cfg.RequestCount != 3 {
					t.Fatalf("RequestCount = %d, want 3", cfg.RequestCount)
				}
			},
		},
		{
			name: "overrides",
			env: map[string]string{
				"MERIDIAN_ROLE":                   "inventory",
				"MERIDIAN_SERVICE_NAME":           "inventory-api",
				"MERIDIAN_RUN_ID":                 "run-456",
				"MERIDIAN_DISABLE_TRACES":         "true",
				"MERIDIAN_HEARTBEAT_INTERVAL":     "2s",
				"MERIDIAN_EXPECT_CHECKOUT_STATUS": "502",
			},
			assert: func(t *testing.T, cfg Config) {
				t.Helper()
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}
			tt.assert(t, cfg)
		})
	}
}

func TestEnvHelpers(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "envBool falls back on invalid",
			run: func(t *testing.T) {
				t.Setenv("MERIDIAN_DISABLE_METRICS", "not-a-bool")
				if got := envBool("MERIDIAN_DISABLE_METRICS", true); !got {
					t.Fatal("envBool() = false, want fallback true")
				}
			},
		},
		{
			name: "envInt falls back on invalid",
			run: func(t *testing.T) {
				t.Setenv("MERIDIAN_TRAFFICGEN_REQUESTS", "not-a-number")
				if got := envInt("MERIDIAN_TRAFFICGEN_REQUESTS", 7); got != 7 {
					t.Fatalf("envInt() = %d, want 7", got)
				}
			},
		},
		{
			name: "envDuration falls back on invalid",
			run: func(t *testing.T) {
				t.Setenv("MERIDIAN_HEARTBEAT_INTERVAL", "not-a-duration")
				if got := envDuration("MERIDIAN_HEARTBEAT_INTERVAL", 9*time.Second); got != 9*time.Second {
					t.Fatalf("envDuration() = %s, want 9s", got)
				}
			},
		},
		{
			name: "envOrDefault uses fallback",
			run: func(t *testing.T) {
				t.Setenv("MERIDIAN_FAKE", "")
				if got := envOrDefault("MERIDIAN_FAKE", "value"); got != "value" {
					t.Fatalf("envOrDefault() = %q, want value", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
