package auth

import (
	"errors"
	"net/url"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
)

var ErrInvalidClerkConfig = errors.New("invalid Clerk configuration")

type ClerkConfig struct {
	SecretKey         string
	AuthorizedParties []string
}

func (c ClerkConfig) Validate() error {
	if strings.TrimSpace(c.SecretKey) == "" {
		return ErrInvalidClerkConfig
	}
	if len(c.AuthorizedParties) == 0 {
		return ErrInvalidClerkConfig
	}
	for _, party := range c.AuthorizedParties {
		parsed, err := url.ParseRequestURI(strings.TrimSpace(party))
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Path != "" {
			return ErrInvalidClerkConfig
		}
	}
	return nil
}

func ConfigureClerk(config ClerkConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}
	clerk.SetKey(strings.TrimSpace(config.SecretKey))
	return nil
}
