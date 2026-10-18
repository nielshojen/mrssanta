package xsrf

import (
	"context"
	"fmt"
	"os"
)

var (
	validAPIKey string
	secret      []byte
)

func Init(ctx context.Context) error {
	validAPIKey = os.Getenv("API_KEY")
	if validAPIKey == "" {
		return fmt.Errorf("missing API_KEY environment variable")
	}

	xsrfSecret := os.Getenv("XSRF_SECRET")
	if xsrfSecret == "" {
		return fmt.Errorf("missing XSRF_SECRET environment variable")
	}
	secret = []byte(xsrfSecret)

	return nil
}
