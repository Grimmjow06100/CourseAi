package middlewares

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	windowStartedAt time.Time
	count           int
}

type UserRateLimiter struct {
	mu       sync.Mutex
	requests int
	window   time.Duration
	entries  map[string]rateLimitEntry
	now      func() time.Time
}

func NewUserRateLimiter(requests int, window time.Duration) (*UserRateLimiter, error) {
	if requests <= 0 {
		return nil, fmt.Errorf("rate limit requests must be positive")
	}
	if window <= 0 {
		return nil, fmt.Errorf("rate limit window must be positive")
	}
	return &UserRateLimiter{
		requests: requests,
		window:   window,
		entries:  make(map[string]rateLimitEntry),
		now:      time.Now,
	}, nil
}

func (l *UserRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		key := c.ClientIP()
		if principal, ok := contract.PrincipalFromContext(c.Request.Context()); ok {
			key = principal.UserID
		}
		allowed, retryAfter := l.allow(key)
		if !allowed {
			seconds := int(retryAfter.Round(time.Second).Seconds())
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			AbortWithError(c, TooManyRequests("too many generation commands", nil))
			return
		}
		c.Next()
	}
}

func (l *UserRateLimiter) allow(key string) (bool, time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.entries[key]
	if !exists || now.Sub(entry.windowStartedAt) >= l.window {
		l.entries[key] = rateLimitEntry{windowStartedAt: now, count: 1}
		l.deleteExpiredEntries(now)
		return true, 0
	}
	if entry.count >= l.requests {
		return false, l.window - now.Sub(entry.windowStartedAt)
	}
	entry.count++
	l.entries[key] = entry
	return true, 0
}

func (l *UserRateLimiter) deleteExpiredEntries(now time.Time) {
	for key, entry := range l.entries {
		if now.Sub(entry.windowStartedAt) >= 2*l.window {
			delete(l.entries, key)
		}
	}
}
