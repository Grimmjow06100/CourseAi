package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type EnvParsable interface {
	~string | ~int | ~bool | ~float64 | time.Duration
}

func Load() error {
	if err := loadDotEnv(".env"); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}
	return nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid line %d", lineNumber)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if key == "" {
			return fmt.Errorf("empty key at line %d", lineNumber)
		}

		value = strings.Trim(value, `"'`)

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env %s: %w", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
func GetEnv[T EnvParsable](key string) (T, error) {
	var zero T
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return zero, fmt.Errorf("env %s: value is missing", key)
	}
	return parseEnvValue[T](key, value)
}

func GetEnvWithDefault[T EnvParsable](key string, fallback T) (T, error) {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return fallback, nil
	}
	return parseEnvValue[T](key, value)
}

func parseEnvValue[T EnvParsable](key, value string) (T, error) {
	var zero T
	var target T
	var anyVal any = &target

	switch ptr := anyVal.(type) {
	case *string:
		*ptr = value
	case *int:
		v, err := strconv.Atoi(value)
		if err != nil {
			return zero, fmt.Errorf("env %s: parse %q as int: %w", key, value, err)
		}
		*ptr = v
	case *bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return zero, fmt.Errorf("env %s: parse %q as bool: %w", key, value, err)
		}
		*ptr = v
	case *float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return zero, fmt.Errorf("env %s: parse %q as float64: %w", key, value, err)
		}
		*ptr = v
	case *time.Duration:
		v, err := time.ParseDuration(value)
		if err != nil {
			return zero, fmt.Errorf("env %s: parse %q as duration: %w", key, value, err)
		}
		*ptr = v
	}

	return target, nil
}
