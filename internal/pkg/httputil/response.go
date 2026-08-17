package httputil

import (
	"encoding/json"
	"math"
	"net/http"
	"reflect"
	"strconv"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

type APIResponse struct {
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Data       any         `json:"data"`
	Error      any         `json:"error"`
	Pagination *Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type PageParams struct {
	Page     int
	PageSize int
}

func ParsePageParams(r *http.Request) PageParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return PageParams{Page: page, PageSize: pageSize}
}

func NewPagination(page, pageSize int, total int64) *Pagination {
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	return &Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

func Offset(page, pageSize int) int {
	return (page - 1) * pageSize
}

func write(w http.ResponseWriter, status int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func normalizeData(data any) any {
	if data == nil {
		return nil
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Slice && val.IsNil() {
		return reflect.MakeSlice(val.Type(), 0, 0).Interface()
	}
	return data
}

func OK(w http.ResponseWriter, message string, data any) {
	write(w, http.StatusOK, APIResponse{
		Success: true, Message: message, Data: normalizeData(data),
		Error: nil, Pagination: nil,
	})
}

func OKWithPagination(w http.ResponseWriter, message string, data any, pagination *Pagination) {
	write(w, http.StatusOK, APIResponse{
		Success: true, Message: message, Data: normalizeData(data),
		Error: nil, Pagination: pagination,
	})
}

func Created(w http.ResponseWriter, message string, data any) {
	write(w, http.StatusCreated, APIResponse{
		Success: true, Message: message, Data: normalizeData(data),
		Error: nil, Pagination: nil,
	})
}

func Fail(w http.ResponseWriter, status int, message string, errDetail any) {
	write(w, status, APIResponse{
		Success: false, Message: message, Data: nil,
		Error: errDetail, Pagination: nil,
	})
}

func NotFound(w http.ResponseWriter, message string) {
	Fail(w, http.StatusNotFound, message, "resource not found")
}

func MethodNotAllowed(w http.ResponseWriter, method string) {
	Fail(w, http.StatusMethodNotAllowed, "method not allowed", method+" is not supported for this endpoint")
}

func Unauthorized(w http.ResponseWriter, message, detail string) {
	Fail(w, http.StatusUnauthorized, message, detail)
}

func Forbidden(w http.ResponseWriter, message, detail string) {
	Fail(w, http.StatusForbidden, message, detail)
}

func InternalError(w http.ResponseWriter, message string, errDetail any) {
	Fail(w, http.StatusInternalServerError, message, errDetail)
}

func Decode(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

// IsJSONRequest reports whether the client expects a JSON API response.
func IsJSONRequest(r *http.Request) bool {
	if r.URL.Path == "/docs" || r.URL.Path == "/openapi.yaml" {
		return false
	}
	return true
}
