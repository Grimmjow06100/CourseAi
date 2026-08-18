package openai

import (
	"errors"
	"net/http"
	"testing"
	"time"

	openaisdk "github.com/openai/openai-go/v3"
)

type testRetryableError interface {
	Retryable() bool
}

type testRetryAfterError interface {
	RetryAfter() time.Duration
}

func TestClassifyProviderRequestError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		retryAfter string
		retryable  bool
		wantDelay  time.Duration
	}{
		{name: "authentication failure", status: http.StatusUnauthorized, retryable: false},
		{name: "rate limit", status: http.StatusTooManyRequests, retryAfter: "30", retryable: true, wantDelay: 30 * time.Second},
		{name: "server failure", status: http.StatusServiceUnavailable, retryable: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cause := newTestAPIError(t, test.status, test.retryAfter)
			classified := classifyProviderRequestError(cause)

			var retryable testRetryableError
			if !errors.As(classified, &retryable) || retryable.Retryable() != test.retryable {
				t.Fatalf("retryable classification = %v, want %v", retryable, test.retryable)
			}
			if !errors.Is(classified, cause) {
				t.Fatal("classified error does not preserve its cause")
			}

			var retryAfter testRetryAfterError
			if !errors.As(classified, &retryAfter) {
				t.Fatal("classified error does not expose RetryAfter")
			}
			if delay := retryAfter.RetryAfter(); delay != test.wantDelay {
				t.Fatalf("retry delay = %s, want %s", delay, test.wantDelay)
			}
		})
	}
}

func TestInvalidModelOutputIsRetryable(t *testing.T) {
	t.Parallel()

	wrapped := errors.Join(errors.New("decode response"), ErrInvalidModelOutput)
	var retryable testRetryableError
	if !errors.As(wrapped, &retryable) || !retryable.Retryable() {
		t.Fatal("invalid model output must be retryable")
	}
}

func TestProviderErrorMetadataAndRetryAfterFormats(t *testing.T) {
	t.Parallel()

	sentinel := retryableSentinel{message: "retry", code: "retry_code"}
	if sentinel.Error() != "retry" || sentinel.ErrorCode() != "retry_code" || !sentinel.Retryable() {
		t.Fatalf("unexpected sentinel metadata: %+v", sentinel)
	}

	now := time.Date(2026, time.August, 13, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		header http.Header
		want   time.Duration
	}{
		{name: "milliseconds", header: http.Header{"Retry-After-Ms": []string{"250.5"}}, want: 250500 * time.Microsecond},
		{name: "HTTP date", header: http.Header{"Retry-After": []string{now.Add(2 * time.Minute).Format(http.TimeFormat)}}, want: 2 * time.Minute},
		{name: "negative is clamped", header: http.Header{"Retry-After": []string{"-5"}}, want: 0},
		{name: "malformed", header: http.Header{"Retry-After": []string{"later"}}, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := providerRetryAfter(&http.Response{Header: test.header}, now); got != test.want {
				t.Fatalf("providerRetryAfter() = %s, want %s", got, test.want)
			}
		})
	}
	if got := classifyProviderRequestError(errors.New("network")); got.Error() != "network" {
		t.Fatalf("non-provider error changed: %v", got)
	}
}

func newTestAPIError(t *testing.T, status int, retryAfter string) *openaisdk.Error {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/responses", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	header := make(http.Header)
	if retryAfter != "" {
		header.Set("Retry-After", retryAfter)
	}
	response := &http.Response{StatusCode: status, Header: header, Request: request}
	return &openaisdk.Error{
		Code:       http.StatusText(status),
		Message:    http.StatusText(status),
		StatusCode: status,
		Request:    request,
		Response:   response,
	}
}
