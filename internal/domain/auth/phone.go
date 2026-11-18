package auth

import (
	"regexp"
	"strings"
)

var (
	phoneNoise  = regexp.MustCompile(`[\s.\-()]`)
	phoneFormat = regexp.MustCompile(`^0[1-9][0-9]{8,9}$`)
)

// NormalizePhone turns a Vietnamese phone number into one canonical form so
// "0912 345 678", "+84912345678" and "84-912-345-678" all match the same
// account: digits only, starting with 0 (10–11 digits). ok is false when s is
// not a phone number (e.g. a username).
func NormalizePhone(s string) (phone string, ok bool) {
	p := phoneNoise.ReplaceAllString(strings.TrimSpace(s), "")
	switch {
	case strings.HasPrefix(p, "+84"):
		p = "0" + p[3:]
	case strings.HasPrefix(p, "84") && len(p) >= 11:
		p = "0" + p[2:]
	}
	if !phoneFormat.MatchString(p) {
		return "", false
	}
	return p, true
}
