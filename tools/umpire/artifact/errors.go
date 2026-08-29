package artifact

import (
	"errors"
)

type ErrorCode string

const (
	ErrorByteLimit          ErrorCode = "byte-limit"
	ErrorSyntax             ErrorCode = "syntax"
	ErrorTokenLimit         ErrorCode = "token-limit"
	ErrorDepthLimit         ErrorCode = "depth-limit"
	ErrorDuplicateKey       ErrorCode = "duplicate-key"
	ErrorCaseCollision      ErrorCode = "case-collision"
	ErrorUnsupportedFormat  ErrorCode = "unsupported-format"
	ErrorWrongFamily        ErrorCode = "wrong-family"
	ErrorUnknownField       ErrorCode = "unknown-field"
	ErrorCollectionLimit    ErrorCode = "collection-limit"
	ErrorStringLimit        ErrorCode = "string-limit"
	ErrorPayloadLimit       ErrorCode = "payload-limit"
	ErrorMalformedValue     ErrorCode = "malformed-value"
	ErrorNoncanonical       ErrorCode = "noncanonical"
	ErrorProvenanceChecksum ErrorCode = "provenance-checksum"
	ErrorArtifactChecksum   ErrorCode = "artifact-checksum"
	ErrorClosure            ErrorCode = "closure"
)

var (
	ErrByteLimit          = &AdmissionError{Code: ErrorByteLimit}
	ErrSyntax             = &AdmissionError{Code: ErrorSyntax}
	ErrTokenLimit         = &AdmissionError{Code: ErrorTokenLimit}
	ErrDepthLimit         = &AdmissionError{Code: ErrorDepthLimit}
	ErrDuplicateKey       = &AdmissionError{Code: ErrorDuplicateKey}
	ErrCaseCollision      = &AdmissionError{Code: ErrorCaseCollision}
	ErrUnsupportedFormat  = &AdmissionError{Code: ErrorUnsupportedFormat}
	ErrWrongFamily        = &AdmissionError{Code: ErrorWrongFamily}
	ErrUnknownField       = &AdmissionError{Code: ErrorUnknownField}
	ErrCollectionLimit    = &AdmissionError{Code: ErrorCollectionLimit}
	ErrStringLimit        = &AdmissionError{Code: ErrorStringLimit}
	ErrPayloadLimit       = &AdmissionError{Code: ErrorPayloadLimit}
	ErrMalformedValue     = &AdmissionError{Code: ErrorMalformedValue}
	ErrNoncanonical       = &AdmissionError{Code: ErrorNoncanonical}
	ErrProvenanceChecksum = &AdmissionError{Code: ErrorProvenanceChecksum}
	ErrArtifactChecksum   = &AdmissionError{Code: ErrorArtifactChecksum}
	ErrClosure            = &AdmissionError{Code: ErrorClosure}
)

type AdmissionError struct {
	Code  ErrorCode
	cause error
}

func (e *AdmissionError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause == nil {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.cause.Error()
}

func (e *AdmissionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *AdmissionError) Is(target error) bool {
	other, ok := target.(*AdmissionError)
	return ok && e != nil && other != nil && e.Code == other.Code
}

func CodeOf(err error) (ErrorCode, bool) {
	var admission *AdmissionError
	if !errors.As(err, &admission) {
		return "", false
	}
	return admission.Code, true
}

func wrapAdmission(code ErrorCode, cause error) error {
	return &AdmissionError{Code: code, cause: cause}
}
