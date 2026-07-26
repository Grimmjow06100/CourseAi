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
func Load() (error) {
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
	valStr := os.Getenv(key)
	if valStr == "" {
		return zero, fmt.Errorf("env %s : la valeur est vide ", key)
	}

	var target T
	var anyVal any = &target

	switch ptr := anyVal.(type) {
	case *string:
		*ptr = valStr
	case *int:
		v, err := strconv.Atoi(valStr)
		if err != nil {
			return zero, fmt.Errorf("env %s: impossible de parser %q en int: %w", key, valStr, err)
		}
		*ptr = v
	case *bool:
		v, err := strconv.ParseBool(valStr)
		if err != nil {
			return zero, fmt.Errorf("env %s: impossible de parser %q en bool: %w", key, valStr, err)
		}
		*ptr = v
	case *float64:
		v, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return zero, fmt.Errorf("env %s: impossible de parser %q en float64: %w", key, valStr, err)
		}
		*ptr = v
	case *time.Duration:
		v, err := time.ParseDuration(valStr)
		if err != nil {
			return zero, fmt.Errorf("env %s: impossible de parser %q en duration: %w", key, valStr, err)
		}
		*ptr = v
	}

	return target, nil
}
func GetEnvWithDefault[T EnvParsable](key string,fallback T) (T, error) {
	
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback, fmt.Errorf("env %s : la valeur est vide ", key)
	}

	var target T
	var anyVal any = &target

	switch ptr := anyVal.(type) {
	case *string:
		*ptr = valStr
	case *int:
		v, err := strconv.Atoi(valStr)
		if err != nil {
			return fallback, fmt.Errorf("env %s: impossible de parser %q en int: %w", key, valStr, err)
		}
		*ptr = v
	case *bool:
		v, err := strconv.ParseBool(valStr)
		if err != nil {
			return fallback, fmt.Errorf("env %s: impossible de parser %q en bool: %w", key, valStr, err)
		}
		*ptr = v
	case *float64:
		v, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return fallback, fmt.Errorf("env %s: impossible de parser %q en float64: %w", key, valStr, err)
		}
		*ptr = v
	case *time.Duration:
		v, err := time.ParseDuration(valStr)
		if err != nil {
			return fallback, fmt.Errorf("env %s: impossible de parser %q en duration: %w", key, valStr, err)
		}
		*ptr = v
	}

	return target, nil
}