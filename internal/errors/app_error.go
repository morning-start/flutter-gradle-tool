package errors

import "fmt"

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return e.Message
	}
	if e.Message == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func Wrap(code int, message string, err error) error {
	return &AppError{Code: code, Message: message, Err: err}
}

func New(code int, message string) error {
	return &AppError{Code: code, Message: message}
}

func Code(err error) int {
	if err == nil {
		return ExitOK
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ExitUnknownCommand
}
