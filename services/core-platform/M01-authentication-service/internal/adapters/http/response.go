package http

import (
	"encoding/json"
	"net/http"
)

type apiError struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeSuccess(w http.ResponseWriter, statusCode int, data any) {
	writeJSON(w, statusCode, map[string]any{
		"status": "success",
		"data":   data,
	})
}

func writeMessage(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]any{
		"status":  "success",
		"message": message,
	})
}

func writeError(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, apiError{
		Status:  "error",
		Code:    code,
		Message: message,
	})
}
