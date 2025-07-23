package services

import "errors"

var (
	// ErrDecisionTimeout is returned when waiting for a user decision times out
	ErrDecisionTimeout = errors.New("timeout waiting for user decision")
	
	// ErrTaskNotFound is returned when a task cannot be found
	ErrTaskNotFound = errors.New("task not found")
	
	// ErrTaskNotActionable is returned when trying to take action on a non-actionable task
	ErrTaskNotActionable = errors.New("task is not actionable")
	
	// ErrInvalidAction is returned when an invalid action is provided
	ErrInvalidAction = errors.New("invalid action type")
)