package errors

import (
	"net/http"
	"reflect"

	"github.com/sirupsen/logrus"
)

type AppError struct {
	StatusCode int
	Message    string
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

func NewBadRequestError(message string) *AppError {
	return NewAppError(http.StatusBadRequest, message)
}

func NewUnauthorizedError(message ...string) *AppError {
	if len(message) > 0 {
		return NewAppError(http.StatusUnauthorized, message[0])
	}
	return NewAppError(http.StatusUnauthorized, "Unauthorized")
}

func NewNotFoundError(message string) *AppError {
	return NewAppError(http.StatusNotFound, message)
}

func NewInternalServerError(originalError error, message string) *AppError {
	logrus.Errorf("[%s] %s", reflect.TypeOf(originalError).String(), originalError)
	return NewAppError(http.StatusInternalServerError, message)
}
