package errors

import (
	"net/http"
	"reflect"

	"github.com/sirupsen/logrus"
)

type AppError struct {
	StatusCode int
	Message    string
	Data       any
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(statusCode int, message string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
	}
}

func NewAppErrorWithData(statusCode int, message string, data any) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	}
}

func NewBadRequestError(message string) *AppError {
	return NewAppError(http.StatusBadRequest, message)
}

func NewUnauthorizedError(message ...string) *AppError {
	if len(message) > 0 {
		return NewAppError(http.StatusUnauthorized, message[0])
	}
	return NewAppError(http.StatusUnauthorized, "Unauthorized")
}

func NewForbiddenError(message string) *AppError {
	return NewAppError(http.StatusForbidden, message)
}

func NewNotFoundError(message string) *AppError {
	return NewAppError(http.StatusNotFound, message)
}

func NewTooManyRequestsError(message string, limit int, resetTime int64) *AppError {
	return NewAppErrorWithData(http.StatusTooManyRequests, message, map[string]interface{}{
		"limit": limit,
		"reset": resetTime,
	})
}

func NewInternalServerError(originalError error, message string) *AppError {
	logrus.Errorf("[%s] %s", reflect.TypeOf(originalError).String(), originalError)
	return NewAppError(http.StatusInternalServerError, message)
}
