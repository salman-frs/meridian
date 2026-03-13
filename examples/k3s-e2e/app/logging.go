package e2eapp

import (
	"encoding/json"
	"os"
	"time"
)

func logEvent(cfg Config, level string, event string, fields map[string]any) {
	logEventInternal(cfg, level, event, fields, false)
}

func logForced(cfg Config, level string, event string, fields map[string]any) {
	logEventInternal(cfg, level, event, fields, true)
}

func logEventInternal(cfg Config, level string, event string, fields map[string]any, force bool) {
	if cfg.DisableAppLogs && !force {
		return
	}

	entry := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     level,
		"event":     event,
		"service":   cfg.ServiceName,
		"role":      cfg.Role,
		"scenario":  cfg.Scenario,
		"run_id":    cfg.RunID,
	}
	for key, value := range fields {
		entry[key] = value
	}

	encoder := json.NewEncoder(os.Stdout)
	_ = encoder.Encode(entry)
}
