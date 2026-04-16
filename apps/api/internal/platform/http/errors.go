package httpx

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorCode string

const (
	CodeInvalidRequest     ErrorCode = "invalid_request"
	CodeValidationFailed   ErrorCode = "validation_failed"
	CodeUnauthorized       ErrorCode = "unauthorized"
	CodeForbidden          ErrorCode = "forbidden"
	CodeNotFound           ErrorCode = "not_found"
	CodeConflict           ErrorCode = "conflict"
	CodeRateLimited        ErrorCode = "rate_limited"
	CodeNotImplemented     ErrorCode = "not_implemented"
	CodeServiceUnavailable ErrorCode = "service_unavailable"
	CodeInternal           ErrorCode = "internal_error"
)

type AppError struct {
	Status  int
	Code    ErrorCode
	Message string
	Err     error
	Details map[string]any
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewError(status int, code ErrorCode, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func WrapError(status int, code ErrorCode, message string, err error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Err: err}
}

func ValidationError(message string, details map[string]any) *AppError {
	return &AppError{
		Status:  http.StatusUnprocessableEntity,
		Code:    CodeValidationFailed,
		Message: message,
		Details: details,
	}
}

func Unauthorized(message string) *AppError {
	return NewError(http.StatusUnauthorized, CodeUnauthorized, message)
}

func Forbidden(message string) *AppError {
	return NewError(http.StatusForbidden, CodeForbidden, message)
}

func NotImplemented(message string) *AppError {
	return NewError(http.StatusNotImplemented, CodeNotImplemented, message)
}

func ServiceUnavailable(message string, err error) *AppError {
	return WrapError(http.StatusServiceUnavailable, CodeServiceUnavailable, message, err)
}

func ToAppError(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return WrapError(http.StatusInternalServerError, CodeInternal, "Внутренняя ошибка сервера", err)
}
