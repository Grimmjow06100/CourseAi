package jobs

import (
	"errors"
	"fmt"
	"strings"
	"time"

	appconfig "github.com/Grimmjow06100/course-ai/backend-go/internal/config"
)

const (
	defaultWorkerConcurrency   = 1
	defaultPollInterval        = time.Second
	defaultLeaseDuration       = 5 * time.Minute
	defaultHeartbeatInterval   = time.Minute
	defaultJobTimeout          = 30 * time.Minute
	defaultShutdownTimeout     = 20 * time.Second
	defaultCleanupTimeout      = 5 * time.Second
	defaultReaperInterval      = 30 * time.Second
	defaultMaxAttempts         = 3
	defaultRetryBaseDelay      = 5 * time.Second
	defaultRetryMaxDelay       = 2 * time.Minute
	defaultRetryJitterFraction = 0.20
	defaultWorkerIDPrefix      = "course-ai"
)

var ErrInvalidWorkerConfig = errors.New("invalid generation worker config")

// WorkerConfig controls the bounded worker pool and the lifecycle of claimed jobs.
type WorkerConfig struct {
	Enabled             bool
	Concurrency         int
	PollInterval        time.Duration
	LeaseDuration       time.Duration
	HeartbeatInterval   time.Duration
	JobTimeout          time.Duration
	ShutdownTimeout     time.Duration
	CleanupTimeout      time.Duration
	ReaperInterval      time.Duration
	DefaultMaxAttempts  int
	RetryBaseDelay      time.Duration
	RetryMaxDelay       time.Duration
	RetryJitterFraction float64
	WorkerIDPrefix      string
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		Enabled:             true,
		Concurrency:         defaultWorkerConcurrency,
		PollInterval:        defaultPollInterval,
		LeaseDuration:       defaultLeaseDuration,
		HeartbeatInterval:   defaultHeartbeatInterval,
		JobTimeout:          defaultJobTimeout,
		ShutdownTimeout:     defaultShutdownTimeout,
		CleanupTimeout:      defaultCleanupTimeout,
		ReaperInterval:      defaultReaperInterval,
		DefaultMaxAttempts:  defaultMaxAttempts,
		RetryBaseDelay:      defaultRetryBaseDelay,
		RetryMaxDelay:       defaultRetryMaxDelay,
		RetryJitterFraction: defaultRetryJitterFraction,
		WorkerIDPrefix:      defaultWorkerIDPrefix,
	}
}

// LoadWorkerConfig reads worker settings from the environment and rejects malformed values.
func LoadWorkerConfig() (WorkerConfig, error) {
	defaults := DefaultWorkerConfig()
	var cfg WorkerConfig
	var err error

	if cfg.Enabled, err = workerConfigValue("GENERATION_WORKER_ENABLED", defaults.Enabled); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.Concurrency, err = workerConfigValue("GENERATION_WORKER_CONCURRENCY", defaults.Concurrency); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.PollInterval, err = workerConfigValue("GENERATION_JOB_POLL_INTERVAL", defaults.PollInterval); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.LeaseDuration, err = workerConfigValue("GENERATION_JOB_LEASE_DURATION", defaults.LeaseDuration); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.HeartbeatInterval, err = workerConfigValue("GENERATION_JOB_HEARTBEAT_INTERVAL", defaults.HeartbeatInterval); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.JobTimeout, err = workerConfigValue("GENERATION_JOB_DEFAULT_TIMEOUT", defaults.JobTimeout); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.ShutdownTimeout, err = workerConfigValue("GENERATION_JOB_SHUTDOWN_TIMEOUT", defaults.ShutdownTimeout); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.CleanupTimeout, err = workerConfigValue("GENERATION_JOB_CLEANUP_TIMEOUT", defaults.CleanupTimeout); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.ReaperInterval, err = workerConfigValue("GENERATION_JOB_REAPER_INTERVAL", defaults.ReaperInterval); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.DefaultMaxAttempts, err = workerConfigValue("GENERATION_JOB_MAX_ATTEMPTS", defaults.DefaultMaxAttempts); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.RetryBaseDelay, err = workerConfigValue("GENERATION_JOB_RETRY_BASE_DELAY", defaults.RetryBaseDelay); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.RetryMaxDelay, err = workerConfigValue("GENERATION_JOB_RETRY_MAX_DELAY", defaults.RetryMaxDelay); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.RetryJitterFraction, err = workerConfigValue("GENERATION_JOB_RETRY_JITTER_FRACTION", defaults.RetryJitterFraction); err != nil {
		return WorkerConfig{}, err
	}
	if cfg.WorkerIDPrefix, err = workerConfigValue("GENERATION_WORKER_ID_PREFIX", defaults.WorkerIDPrefix); err != nil {
		return WorkerConfig{}, err
	}
	cfg.WorkerIDPrefix = strings.TrimSpace(cfg.WorkerIDPrefix)

	if err := cfg.Validate(); err != nil {
		return WorkerConfig{}, err
	}
	return cfg, nil
}

func (c WorkerConfig) Validate() error {
	checks := []struct {
		invalid bool
		field   string
		value   any
	}{
		{c.Concurrency <= 0, "concurrency", c.Concurrency},
		{c.PollInterval <= 0, "poll interval", c.PollInterval},
		{c.LeaseDuration <= 0, "lease duration", c.LeaseDuration},
		{c.HeartbeatInterval <= 0, "heartbeat interval", c.HeartbeatInterval},
		{c.JobTimeout <= 0, "job timeout", c.JobTimeout},
		{c.ShutdownTimeout <= 0, "shutdown timeout", c.ShutdownTimeout},
		{c.CleanupTimeout <= 0, "cleanup timeout", c.CleanupTimeout},
		{c.ReaperInterval <= 0, "reaper interval", c.ReaperInterval},
		{c.DefaultMaxAttempts <= 0, "default max attempts", c.DefaultMaxAttempts},
		{c.RetryBaseDelay <= 0, "retry base delay", c.RetryBaseDelay},
		{c.RetryMaxDelay <= 0, "retry max delay", c.RetryMaxDelay},
		{c.RetryJitterFraction < 0 || c.RetryJitterFraction > 1, "retry jitter fraction", c.RetryJitterFraction},
		{strings.TrimSpace(c.WorkerIDPrefix) == "", "worker id prefix", c.WorkerIDPrefix},
	}
	for _, check := range checks {
		if check.invalid {
			return fmt.Errorf("%w: %s=%v", ErrInvalidWorkerConfig, check.field, check.value)
		}
	}

	if c.HeartbeatInterval >= c.LeaseDuration/2 {
		return fmt.Errorf("%w: heartbeat interval must be less than half the lease duration", ErrInvalidWorkerConfig)
	}
	if c.JobTimeout <= c.HeartbeatInterval {
		return fmt.Errorf("%w: job timeout must exceed the heartbeat interval", ErrInvalidWorkerConfig)
	}
	if c.CleanupTimeout > c.ShutdownTimeout {
		return fmt.Errorf("%w: cleanup timeout must not exceed shutdown timeout", ErrInvalidWorkerConfig)
	}
	if c.ReaperInterval > c.LeaseDuration {
		return fmt.Errorf("%w: reaper interval must not exceed lease duration", ErrInvalidWorkerConfig)
	}
	if c.RetryMaxDelay < c.RetryBaseDelay {
		return fmt.Errorf("%w: retry max delay must be greater than or equal to retry base delay", ErrInvalidWorkerConfig)
	}
	return nil
}

func workerConfigValue[T appconfig.EnvParsable](key string, fallback T) (T, error) {
	value, err := appconfig.GetEnvWithDefault(key, fallback)
	if err != nil {
		return fallback, fmt.Errorf("%w: %w", ErrInvalidWorkerConfig, err)
	}
	return value, nil
}
