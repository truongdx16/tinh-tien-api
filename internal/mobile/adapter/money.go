// Package adapter provides formatting helpers for the Flutter mobile contract.
// Flutter expects money as decimal strings, status as int, pagination with current_page.
package adapter

import (
	"fmt"
	"strconv"
	"strings"
)

// MoneyStr formats an int64 VND value as a plain numeric string (e.g. "15000").
func MoneyStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

// FloatStr formats a float64 (quantity) as a plain numeric string without trailing zeros.
func FloatStr(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	return s
}

// ParseMoney parses a decimal string (possibly float like "15000.5") to int64 VND.
func ParseMoney(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(f), nil
	}
	return 0, fmt.Errorf("invalid money value: %s", s)
}

// BoolToStatus converts a bool active flag to Flutter int (1=active, 0=inactive).
func BoolToStatus(active bool) int {
	if active {
		return 1
	}
	return 0
}

// StatusToBool converts Flutter int status (1=active) to bool.
func StatusToBool(status int) bool {
	return status == 1
}
