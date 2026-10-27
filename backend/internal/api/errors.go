package api

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{Code: code, Message: message})
}

func writeValidation(w http.ResponseWriter, details map[string]interface{}) {
	writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{
		Code:    "validation",
		Message: "invalid request",
		Details: details,
	})
}
