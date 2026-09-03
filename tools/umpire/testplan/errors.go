package testplan

import (
	"errors"
	"fmt"
)

// ErrorCode identifies the admission invariant that rejected a plan.
type ErrorCode string

const (
	ErrorByteLimit           ErrorCode = "byte-limit"
	ErrorUnknownField        ErrorCode = "unknown-field"
	ErrorUnsupportedVersion  ErrorCode = "unsupported-version"
	ErrorUnsupportedEnum     ErrorCode = "unsupported-enum"
	ErrorUnsupportedOperator ErrorCode = "unsupported-operator"
	ErrorMalformedValue      ErrorCode = "malformed-value"
	ErrorDuplicate           ErrorCode = "duplicate"
	ErrorOrdering            ErrorCode = "ordering"
	ErrorLimit               ErrorCode = "limit"
	ErrorBinding             ErrorCode = "binding"
	ErrorChecksum            ErrorCode = "checksum"
	ErrorProvenance          ErrorCode = "provenance"
	ErrorResultAuthority     ErrorCode = "result-authority"
)

// AdmissionError reports a fail-closed portable plan admission failure.
type AdmissionError struct {
	Code ErrorCode
	Path string
	err  error
}

func (e *AdmissionError) Error() string {
	if e == nil {
		return ""
	}
	detail := string(e.Code)
	if e.Path != "" {
		detail += " at " + e.Path
	}
	if e.err != nil {
		detail += ": " + e.err.Error()
	}
	return detail
}

func (e *AdmissionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// CodeOf returns the stable code for an admission failure.
func CodeOf(err error) (ErrorCode, bool) {
	var admission *AdmissionError
	if !errors.As(err, &admission) {
		return "", false
	}
	return admission.Code, true
}

func admissionError(code ErrorCode, path, format string, args ...any) error {
	return &AdmissionError{Code: code, Path: path, err: fmt.Errorf(format, args...)}
}
