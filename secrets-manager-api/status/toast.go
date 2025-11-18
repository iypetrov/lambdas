package status

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Toast struct {
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}

func (t Toast) Error() string {
	return fmt.Sprintf("custom error: %s", t.Message)
}

func AddToast(w http.ResponseWriter, t Toast) {
	res, err := json.Marshal(struct {
		Toast Toast `json:"add-toast"`
	}{
		Toast: t,
	})
	if err != nil {
		return
	}
	w.Header().Set("HX-Trigger", string(res))
}
