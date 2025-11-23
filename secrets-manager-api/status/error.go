package status

import (
	"fmt"
	"net/http"
)

var (
	ErrParsingFrom = fmt.Errorf("failed to parse a form")
)

func ErrorBadRequest(err error) Toast {
	return Toast{
		Message:    err.Error(),
		StatusCode: http.StatusBadRequest,
	}
}

func ErrorNotFound(err error) Toast {
	return Toast{
		Message:    err.Error(),
		StatusCode: http.StatusNotFound,
	}
}

func ErrorConflict(err error) Toast {
	return Toast{
		Message:    err.Error(),
		StatusCode: http.StatusConflict,
	}
}

func ErrorInternalServerError(err error) Toast {
	return Toast{
		Message:    err.Error(),
		StatusCode: http.StatusInternalServerError,
	}
}
