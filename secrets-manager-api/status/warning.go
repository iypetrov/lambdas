package status

import (
	"fmt"
	"net/http"
)

var (
	WarnNotNumbericID = fmt.Errorf("id should be a number")
)

func WarningStatusBadRequest(err error) Toast {
	return Toast{
		Message:    err.Error(),
		StatusCode: http.StatusBadRequest,
	}
}

func WarningStatunUnauthorized(err error) Toast {
	return Toast{
		Message:    err.Error(),
		StatusCode: http.StatusUnauthorized,
	}
}

func WarningStatusForbidden(err error) Toast {
	return Toast{
		Message:    err.Error(),
		StatusCode: http.StatusForbidden,
	}
}
