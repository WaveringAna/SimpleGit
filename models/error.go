//models/error.go

package models

import (
	config "SimpleGit/config"

	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
)

type ErrorType string

const (
	ErrorTypeNotFound      ErrorType = "NOT_FOUND"
	ErrorTypeUnauthorized  ErrorType = "UNAUTHORIZED"
	ErrorTypeBadRequest    ErrorType = "BAD_REQUEST"
	ErrorTypeInternal      ErrorType = "INTERNAL"
	ErrorTypeGit           ErrorType = "GIT_ERROR"
	ErrorTypeInvalidPath   ErrorType = "INVALID_PATH"
	ErrorTypeInvalidBranch ErrorType = "INVALID_BRANCH"
)

type AppError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Detail     string    `json:"detail,omitempty"`
	Code       int       `json:"-"`
	RequestID  string    `json:"request_id,omitempty"`
	File       string    `json:"file,omitempty"`
	Line       int       `json:"line,omitempty"`
	Err        error     `json:"-"`
	ShowInProd bool      `json:"-"`
}

// Add a method to mark errors as production-safe
func (e *AppError) ShowInProduction() *AppError {
	e.ShowInProd = true
	return e
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap implements the errors.Unwrap interface
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewError creates a new AppError with stack trace
func NewError(errType ErrorType, message string, code int) *AppError {
	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		Type:    errType,
		Message: message,
		Code:    code,
		File:    filepath.Base(file),
		Line:    line,
	}
}

// WithDetail adds detail to the error
func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail
	return e
}

// WithRequestID adds a request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// WithError wraps an underlying error
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	if err != nil {
		e.Detail = err.Error()
	}
	return e
}

// Common error constructors
func NewNotFoundError(message string) *AppError {
	return NewError(ErrorTypeNotFound, message, http.StatusNotFound)
}

func NewUnauthorizedError(message string) *AppError {
	return NewError(ErrorTypeUnauthorized, message, http.StatusUnauthorized)
}

func NewBadRequestError(message string) *AppError {
	return NewError(ErrorTypeBadRequest, message, http.StatusBadRequest)
}

func NewInternalError(message string) *AppError {
	return NewError(ErrorTypeInternal, message, http.StatusInternalServerError)
}

func NewGitError(message string, err error) *AppError {
	return NewError(ErrorTypeGit, message, http.StatusInternalServerError).WithError(err)
}

// HandleError writes the error to the response in a consistent format
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = NewInternalError("Internal Server Error").WithError(err)
	}

	// Build error details for logging
	details := []string{
		fmt.Sprintf("Type: %s", appErr.Type),
		fmt.Sprintf("Message: %s", appErr.Message),
		fmt.Sprintf("Location: %s:%d", appErr.File, appErr.Line),
	}

	if appErr.RequestID != "" {
		details = append(details, fmt.Sprintf("RequestID: %s", appErr.RequestID))
	}
	if appErr.Detail != "" {
		details = append(details, fmt.Sprintf("Detail: %s", appErr.Detail))
	}
	if appErr.Err != nil {
		details = append(details, fmt.Sprintf("Error: %v", appErr.Err))
	}

	// Log all error details
	log.Printf("Error occurred:\n\t%s", strings.Join(details, "\n\t"))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Code)

	if config.GlobalConfig.DevMode || appErr.ShowInProd {
		json.NewEncoder(w).Encode(appErr)
	} else {
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Internal Server Error",
		})
	}
}
