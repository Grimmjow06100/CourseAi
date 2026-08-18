package openai

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	openaisdk "github.com/openai/openai-go/v3"
)

type retryableSentinel struct {
	message string
	code    string
}

func (e retryableSentinel) Error() string {
	return e.message
}

func (e retryableSentinel) ErrorCode() string {
	return e.code
}

func (e retryableSentinel) Retryable() bool {
	return true
}

type providerRequestError struct {
	cause      error
	code       string
	retryable  bool
	retryAfter time.Duration
}

func (e providerRequestError) Error() string {
	return e.cause.Error()
}

func (e providerRequestError) Unwrap() error {
	return e.cause
}

func (e providerRequestError) ErrorCode() string {
	return e.code
}

func (e providerRequestError) Retryable() bool {
	return e.retryable
}

func (e providerRequestError) RetryAfter() time.Duration {
	return e.retryAfter
}

func classifyProviderRequestError(cause error) error {
	var apiError *openaisdk.Error
	if !errors.As(cause, &apiError) {
		return cause
	}

	status := apiError.StatusCode
	if status == 0 && apiError.Response != nil {
		status = apiError.Response.StatusCode
	}
	code := strings.TrimSpace(apiError.Code)
	if code == "" {
		code = "openai_http_" + strconv.Itoa(status)
	}
	return providerRequestError{
		cause:      cause,
		code:       code,
		retryable:  isRetryableProviderStatus(status),
		retryAfter: providerRetryAfter(apiError.Response, time.Now()),
	}
}

func isRetryableProviderStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusConflict ||
		status == http.StatusTooManyRequests ||
		status >= http.StatusInternalServerError
}

func providerRetryAfter(response *http.Response, now time.Time) time.Duration {
	if response == nil {
		return 0
	}
	if milliseconds := strings.TrimSpace(response.Header.Get("retry-after-ms")); milliseconds != "" {
		if value, err := strconv.ParseFloat(milliseconds, 64); err == nil {
			return max(time.Duration(value*float64(time.Millisecond)), 0)
		}
	}

	value := strings.TrimSpace(response.Header.Get("Retry-After"))
	if value == "" {
		return 0
	}
	if seconds, err := strconv.ParseFloat(value, 64); err == nil {
		return max(time.Duration(seconds*float64(time.Second)), 0)
	}
	if date, err := http.ParseTime(value); err == nil {
		return max(date.Sub(now), 0)
	}
	return 0
}
