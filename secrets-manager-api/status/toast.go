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
	// Use HX-Trigger-After-Settle to ensure the event fires after Alpine.js re-initializes
	// This is important when replacing the entire body content
	w.Header().Set("HX-Trigger-After-Settle", string(res))
	// Also set HX-Trigger for cases where we're not replacing the body
	w.Header().Set("HX-Trigger", string(res))
}
