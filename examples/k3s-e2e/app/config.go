package e2eapp

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Role                   string
	ServiceName            string
	Port                   string
	RunID                  string
	Scenario               string
	OTLPEndpoint           string
	OTLPInsecure           bool
	OTLPTimeout            time.Duration
	DisableTraces          bool
	DisableMetrics         bool
	DisableAppLogs         bool
	StorefrontURL          string
	CheckoutURL            string
	InventoryURL           string
	HeartbeatInterval      time.Duration
	RequestCount           int
	RequestDelay           time.Duration
	ExpectedBrowseStatus   int
	ExpectedCheckoutStatus int
	ExpectedToken          string
	OutboundToken          string
}

func LoadConfig() (Config, error) {
	role := envOrDefault("MERIDIAN_ROLE", "storefront")
	serviceName := envOrDefault("MERIDIAN_SERVICE_NAME", role)

	cfg := Config{
		Role:                   role,
		ServiceName:            serviceName,
		Port:                   envOrDefault("PORT", "8080"),
		RunID:                  envOrDefault("MERIDIAN_RUN_ID", "meridian-dev-run"),
		Scenario:               envOrDefault("MERIDIAN_SCENARIO", "happy"),
		OTLPEndpoint:           envOrDefault("MERIDIAN_OTLP_ENDPOINT", "otel-gateway.observability.svc.cluster.local:4317"),
		OTLPInsecure:           envBool("MERIDIAN_OTLP_INSECURE", true),
		OTLPTimeout:            envDuration("MERIDIAN_OTLP_TIMEOUT", 3*time.Second),
		DisableTraces:          envBool("MERIDIAN_DISABLE_TRACES", false),
		DisableMetrics:         envBool("MERIDIAN_DISABLE_METRICS", false),
		DisableAppLogs:         envBool("MERIDIAN_DISABLE_APP_LOGS", false),
		StorefrontURL:          strings.TrimRight(envOrDefault("MERIDIAN_STOREFRONT_URL", "http://storefront:8080"), "/"),
		CheckoutURL:            strings.TrimRight(envOrDefault("MERIDIAN_CHECKOUT_URL", "http://checkout:8080"), "/"),
		InventoryURL:           strings.TrimRight(envOrDefault("MERIDIAN_INVENTORY_URL", "http://inventory:8080"), "/"),
		HeartbeatInterval:      envDuration("MERIDIAN_HEARTBEAT_INTERVAL", 5*time.Second),
		RequestCount:           envInt("MERIDIAN_TRAFFICGEN_REQUESTS", 3),
		RequestDelay:           envDuration("MERIDIAN_TRAFFICGEN_DELAY", 1*time.Second),
		ExpectedBrowseStatus:   envInt("MERIDIAN_EXPECT_BROWSE_STATUS", 200),
		ExpectedCheckoutStatus: envInt("MERIDIAN_EXPECT_CHECKOUT_STATUS", 200),
		ExpectedToken:          envOrDefault("MERIDIAN_EXPECTED_TOKEN", "meridian-shared-secret"),
		OutboundToken:          envOrDefault("MERIDIAN_OUTBOUND_TOKEN", "meridian-shared-secret"),
	}

	if cfg.Role == "" {
		return Config{}, fmt.Errorf("MERIDIAN_ROLE must not be empty")
	}
	if cfg.RunID == "" {
		return Config{}, fmt.Errorf("MERIDIAN_RUN_ID must not be empty")
	}
	if cfg.Port == "" && cfg.Role != "trafficgen" {
		return Config{}, fmt.Errorf("PORT must not be empty for role %q", cfg.Role)
	}
	return cfg, nil
}

func envOrDefault(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
