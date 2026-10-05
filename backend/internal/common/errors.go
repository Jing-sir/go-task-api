package common

type AppError struct {
	Code    int
	Message string
}

func (a AppError) Error() string {
	return a.Message
}

func NotFound(message string) *AppError {
	return &AppError{
		Code:    404,
		Message: message,
	}
}

func BadRequest(message string) *AppError {
	return &AppError{
		Code:    400,
		Message: message,
	}
}

func Unauthorized(message string) *AppError {
	return &AppError{
		Code:    401,
		Message: message,
	}
}

func InternalError(message string) *AppError {
	return &AppError{
		Code:    500,
		Message: message,
	}
}
