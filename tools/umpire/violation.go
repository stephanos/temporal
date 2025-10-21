package umpire

import "time"

// Violation represents a detected invariant violation.
type Violation struct {
	// Model is the name of the model that detected the violation
	Model string

	// Message is a human-readable description of the violation
	Message string

	// Tags contains additional context about the violation
	Tags map[string]string

	// Timestamp is when the violation was detected
	Timestamp time.Time
}
