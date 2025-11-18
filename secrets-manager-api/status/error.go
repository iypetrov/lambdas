package status

import (
	"fmt"
	"net/http"
)

var (
	ErrParsingFrom             = fmt.Errorf("failed to parse a form")
)

func ErrorNotFound(err error) Toast {
	return Toast{
		Message:    err.Error(),
		StatusCode: http.StatusNotFound,
	}
}

func ErrorInternalServerError(err error) Toast {
	return Toast{
		Message:    err.Error(),
		StatusCode: http.StatusInternalServerError,
	}
}
