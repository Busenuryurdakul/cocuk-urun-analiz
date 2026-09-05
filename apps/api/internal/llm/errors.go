package llm

import "errors"

var (
	ErrForbidden            = errors.New("llm access forbidden")
	ErrInvalidInput         = errors.New("invalid llm input")
	ErrModelDisabled        = errors.New("llm model disabled")
	ErrRoutingFailed        = errors.New("llm routing failed")
	ErrPublishValidation    = errors.New("llm publish validation failed")
	ErrCircuitOpen          = errors.New("llm provider circuit open")
	ErrComplianceBlocked    = errors.New("llm call blocked by compliance")
	ErrPromptInjection      = errors.New("llm prompt injection blocked")
	ErrInsufficientModels   = errors.New("insufficient active llm models")
	ErrDraftNotValidated    = errors.New("llm draft not validated")
	ErrInvalidPolicyVersion = errors.New("invalid llm policy version pin")
)
