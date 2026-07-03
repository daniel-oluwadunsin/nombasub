package responses

import "net/http"

type AppError struct {
	StatusCode int
	Message    string
	Data       interface{}
	Err        error
}

func NewAppError(statusCode int, message string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
	}
}

func (e *AppError) Error() string {
	if e.Err != nil {
		message := e.Err.Error()

		if message != "" {
			return message
		}
	}

	return e.Message
}

func (e *AppError) WithData(data interface{}) *AppError {
	e.Data = data
	return e
}

func BadRequest(message string) *AppError {
	return NewAppError(http.StatusBadRequest, message)
}

func NotFound(message string) *AppError {
	return NewAppError(http.StatusNotFound, message)
}

func Unauthorized(message string) *AppError {
	return NewAppError(http.StatusUnauthorized, message)
}

func Forbidden(message string) *AppError {
	return NewAppError(http.StatusForbidden, message)
}

func Conflict(message string) *AppError {
	return NewAppError(http.StatusConflict, message)
}

func InternalServerError(err error) *AppError {
	return &AppError{
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal Server Error",
		Err:        err,
		Data:       err.Error(),
	}
}

func AppErrorFromError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	return InternalServerError(err)
}
