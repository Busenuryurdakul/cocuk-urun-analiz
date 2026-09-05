package agent

import (
	"errors"
	"strings"
	"time"
)

type RetryDecision struct {
	Retryable   bool
	MaxAttempts int
	Backoff     time.Duration
}

type RetryClassifier struct{}

func (RetryClassifier) Classify(err error) RetryDecision {
	defaultNonRetry := RetryDecision{Retryable: false, MaxAttempts: 1}
	if err == nil {
		return defaultNonRetry
	}
	msg := strings.ToLower(err.Error())
	nonRetryable := []error{
		ErrForbidden, ErrInvalidInput, ErrToolUnavailable, ErrAuthorizationDenied,
		ErrGrantExpired, ErrGrantReplay, ErrComplianceRejected, ErrSchemaValidation,
		ErrGrantStoreUnavailable, ErrInvalidTransition,
	}
	for _, sentinel := range nonRetryable {
		if errors.Is(err, sentinel) {
			return defaultNonRetry
		}
	}
	if strings.Contains(msg, "schema") || strings.Contains(msg, "compliance") ||
		strings.Contains(msg, "authorization") || strings.Contains(msg, "grant") ||
		strings.Contains(msg, "forbidden") || strings.Contains(msg, "tenant") {
		return defaultNonRetry
	}
	return RetryDecision{Retryable: true, MaxAttempts: 3, Backoff: 500 * time.Millisecond}
}

func BoundedBackoff(base time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		return base
	}
	d := base * time.Duration(1<<uint(attempt-1))
	if d > 5*time.Second {
		return 5 * time.Second
	}
	return d
}
