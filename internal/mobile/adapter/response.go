package adapter

import (
	"encoding/json"
	"net/http"
)

// MobileEnvelope is the Flutter-compatible response envelope.
type MobileEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PaginatedData wraps a list with current_page as Flutter expects.
type PaginatedData struct {
	Data        interface{} `json:"data"`
	CurrentPage int         `json:"current_page"`
	Total       int64       `json:"total,omitempty"`
	PerPage     int         `json:"per_page,omitempty"`
}

func write(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// OK writes 200 with data wrapped in MobileEnvelope.
func OK(w http.ResponseWriter, msg string, data interface{}) {
	write(w, http.StatusOK, MobileEnvelope{Success: true, Message: msg, Data: data})
}

// Created writes 201 with data wrapped in MobileEnvelope.
func Created(w http.ResponseWriter, msg string, data interface{}) {
	write(w, http.StatusCreated, MobileEnvelope{Success: true, Message: msg, Data: data})
}

// OKPaginated wraps list + pagination info in Flutter expected shape.
func OKPaginated(w http.ResponseWriter, msg string, items interface{}, page int, total int64, perPage int) {
	data := PaginatedData{Data: items, CurrentPage: page, Total: total, PerPage: perPage}
	write(w, http.StatusOK, MobileEnvelope{Success: true, Message: msg, Data: data})
}

// Fail writes an error response.
func Fail(w http.ResponseWriter, status int, msg string) {
	write(w, status, MobileEnvelope{Success: false, Message: msg, Data: nil})
}
