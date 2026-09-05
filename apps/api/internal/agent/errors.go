package agent

import "errors"

var (
	ErrForbidden               = errors.New("forbidden")
	ErrNotFound                = errors.New("not found")
	ErrInvalidInput            = errors.New("invalid input")
	ErrDuplicateRequest        = errors.New("duplicate client request")
	ErrRunNotCancellable       = errors.New("run not cancellable")
	ErrToolUnavailable         = errors.New("tool unavailable")
	ErrAuthorizationDenied     = errors.New("authorization denied")
	ErrGrantExpired            = errors.New("grant expired")
	ErrGrantReplay             = errors.New("grant replay")
	ErrComplianceRejected      = errors.New("compliance rejected")
	ErrNoAvailableTool         = errors.New("no available tool")
	ErrOrchestratorUnavailable = errors.New("orchestrator unavailable")
	ErrInvalidTransition       = errors.New("invalid run state transition")
	ErrSchemaValidation        = errors.New("schema validation failed")
	ErrGrantStoreUnavailable   = errors.New("grant store unavailable")
	ErrRunTerminal             = errors.New("run already terminal")
)
