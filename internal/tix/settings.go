package tix

import "errors"

const (
	optionPrefix      = "prefix"
	optionType        = "type"
	optionPriority    = "priority"
	optionTag         = "tag"
	optionAcceptance  = "acceptance"
	optionDescription = "description"
	optionDraft       = "draft"
	optionJSON        = "json"
	integerPattern    = `^-?\d+$`
	problemPriority   = "Invalid input: expected integer, received "
)

var (
	ErrExpectedOneTitle     = errors.New("expected exactly one title")
	ErrExpectedOneReference = errors.New("expected exactly one ticket reference")
)
