package response

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

type Pagination struct {
	Page      int   `json:"page"`
	Limit     int   `json:"limit"`
	TotalData int64 `json:"total_data"`
	TotalPage int   `json:"total_page"`
}

type PaginatedData struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func Success(w http.ResponseWriter, status int, message string, data interface{}) {
	writeJSON(w, status, Envelope{Success: true, Message: message, Data: data})
}

func SuccessPaginated(w http.ResponseWriter, message string, data interface{}, pagination Pagination) {
	writeJSON(w, http.StatusOK, Envelope{
		Success: true,
		Message: message,
		Data:    PaginatedData{Data: data, Pagination: pagination},
	})
}

func Error(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorEnvelope{Success: false, Message: message})
}

func ValidationError(w http.ResponseWriter, errors map[string][]string) {
	writeJSON(w, http.StatusUnprocessableEntity, ErrorEnvelope{
		Success: false,
		Message: "Validation Error",
		Errors:  errors,
	})
}

func Created(w http.ResponseWriter, message string, data interface{}) {
	Success(w, http.StatusCreated, message, data)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
