package jobs

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestDefaultWorkerConfigIsValid(t *testing.T) {
	t.Parallel()
	if err := DefaultWorkerConfig().Validate(); err != nil {
		t.Fatalf("default config: %v", err)
	}
}

func TestConfigRejectsUnsafeHeartbeat(t *testing.T) {
	t.Parallel()
	cfg := DefaultWorkerConfig()
	cfg.HeartbeatInterval = cfg.LeaseDuration / 2
	if err := cfg.Validate(); !errors.Is(err, ErrInvalidWorkerConfig) {
		t.Fatalf("error = %v, want ErrInvalidWorkerConfig", err)
	}
}

func TestLoadWorkerConfigReadsEnvironmentStrictly(t *testing.T) {
	setWorkerEnvironment(t, map[string]string{
		"GENERATION_WORKER_ENABLED":            "false",
		"GENERATION_WORKER_CONCURRENCY":        "2",
		"GENERATION_JOB_POLL_INTERVAL":         "20ms",
		"GENERATION_JOB_LEASE_DURATION":        "2s",
		"GENERATION_JOB_HEARTBEAT_INTERVAL":    "200ms",
		"GENERATION_JOB_DEFAULT_TIMEOUT":       "4s",
		"GENERATION_JOB_SHUTDOWN_TIMEOUT":      "1s",
		"GENERATION_JOB_CLEANUP_TIMEOUT":       "100ms",
		"GENERATION_JOB_REAPER_INTERVAL":       "500ms",
		"GENERATION_JOB_MAX_ATTEMPTS":          "4",
		"GENERATION_JOB_RETRY_BASE_DELAY":      "30ms",
		"GENERATION_JOB_RETRY_MAX_DELAY":       "1s",
		"GENERATION_JOB_RETRY_JITTER_FRACTION": "0.15",
		"GENERATION_WORKER_ID_PREFIX":          "test-worker",
	})

	cfg, err := LoadWorkerConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Enabled || cfg.Concurrency != 2 || cfg.PollInterval != 20*time.Millisecond || cfg.WorkerIDPrefix != "test-worker" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadWorkerConfigRejectsMalformedEnvironment(t *testing.T) {
	setWorkerEnvironment(t, map[string]string{"GENERATION_WORKER_CONCURRENCY": "many"})
	if _, err := LoadWorkerConfig(); !errors.Is(err, ErrInvalidWorkerConfig) {
		t.Fatalf("error = %v, want ErrInvalidWorkerConfig", err)
	}
}

func setWorkerEnvironment(t *testing.T, values map[string]string) {
	t.Helper()
	keys := []string{
		"GENERATION_WORKER_ENABLED",
		"GENERATION_WORKER_CONCURRENCY",
		"GENERATION_JOB_POLL_INTERVAL",
		"GENERATION_JOB_LEASE_DURATION",
		"GENERATION_JOB_HEARTBEAT_INTERVAL",
		"GENERATION_JOB_DEFAULT_TIMEOUT",
		"GENERATION_JOB_SHUTDOWN_TIMEOUT",
		"GENERATION_JOB_CLEANUP_TIMEOUT",
		"GENERATION_JOB_REAPER_INTERVAL",
		"GENERATION_JOB_MAX_ATTEMPTS",
		"GENERATION_JOB_RETRY_BASE_DELAY",
		"GENERATION_JOB_RETRY_MAX_DELAY",
		"GENERATION_JOB_RETRY_JITTER_FRACTION",
		"GENERATION_WORKER_ID_PREFIX",
	}
	for _, key := range keys {
		previous, existed := os.LookupEnv(key)
		if value, ok := values[key]; ok {
			if err := os.Setenv(key, value); err != nil {
				t.Fatalf("set %s: %v", key, err)
			}
		} else if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(key, previous)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}
