package service

import (
	"errors"
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/response"
)

// ErrorCode is the stable machine-readable classification for warehouse failures.
type ErrorCode string

const (
	// ValidationFailed marks an invalid command value.
	ValidationFailed ErrorCode = "VALIDATION_FAILED"
	// DuplicateValue marks a unique active value conflict.
	DuplicateValue ErrorCode = "DUPLICATE_VALUE"
	// VersionConflict marks an optimistic-concurrency conflict.
	VersionConflict ErrorCode = "VERSION_CONFLICT"
	// StateConflict marks an invalid lifecycle transition.
	StateConflict ErrorCode = "STATE_CONFLICT"
	// ReferenceInUse marks a destructive action blocked by active references.
	ReferenceInUse ErrorCode = "REFERENCE_IN_USE"
	// NotFound marks an unavailable domain record.
	NotFound ErrorCode = "NOT_FOUND"
	// Forbidden marks an authorized-domain action denial.
	Forbidden ErrorCode = "FORBIDDEN"
	// ImportFailed marks a workbook with one or more business errors.
	ImportFailed ErrorCode = "IMPORT_FAILED"
	// NumberExhausted marks an exhausted daily delivery sequence.
	NumberExhausted ErrorCode = "NUMBER_EXHAUSTED"
)

// DomainError carries only safe, client-facing business information. It never
// stores driver errors, SQL text, or other persistence implementation details.
type DomainError struct {
	Code           ErrorCode
	Message        string
	Field          string
	Sheet          string
	LineNo         int
	Column         string
	CurrentVersion uint64
	Errors         []*DomainError
}

// Error implements error without adding persistence details to the client-facing
// message. Returning *DomainError directly keeps errors.As usable through wraps.
func (e *DomainError) Error() string {
	if e.Field == "" {
		return string(e.Code) + ": " + e.Message
	}
	return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Field)
}

// NewDomainError creates a safe business error with a stable code.
func NewDomainError(code ErrorCode, message string) *DomainError {
	return &DomainError{Code: code, Message: message}
}

// WithField associates the error with a writable or query field.
func (e *DomainError) WithField(field string) *DomainError {
	e.Field = field
	return e
}

// WithCurrentVersion annotates a conflict with the authoritative persisted version.
func (e *DomainError) WithCurrentVersion(version uint64) *DomainError {
	e.CurrentVersion = version
	return e
}

// WithExcelLocation associates an import error with its worksheet cell.
func (e *DomainError) WithExcelLocation(sheet string, lineNo int, column string) *DomainError {
	e.Sheet = sheet
	e.LineNo = lineNo
	e.Column = column
	return e
}

// WithErrors attaches safe per-item errors to an aggregate business error.
func (e *DomainError) WithErrors(itemErrors ...*DomainError) *DomainError {
	for _, itemError := range itemErrors {
		if itemError != nil {
			e.Errors = append(e.Errors, itemError)
		}
	}
	return e
}

// AsDomainError extracts a direct or wrapped DomainError using errors.As.
func AsDomainError(err error) (*DomainError, bool) {
	var target *DomainError
	if !errors.As(err, &target) {
		return nil, false
	}
	return target, true
}

// BusinessError converts a domain failure into the uniform response payload.
func (e *DomainError) BusinessError() response.BusinessError {
	result := response.BusinessError{
		ErrorCode:      string(e.Code),
		Message:        e.Message,
		Field:          e.Field,
		Sheet:          e.Sheet,
		LineNo:         e.LineNo,
		Column:         e.Column,
		CurrentVersion: e.CurrentVersion,
	}
	for _, itemError := range e.Errors {
		if itemError != nil {
			result.Errors = append(result.Errors, itemError.BusinessError())
		}
	}
	return result
}

func domainError(code ErrorCode, message, field string) *DomainError {
	return NewDomainError(code, message).WithField(field)
}
