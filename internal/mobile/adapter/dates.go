package adapter

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidDate is returned (wrapped) by DayRange for unparsable from/to.
var ErrInvalidDate = errors.New("invalid date")

// DayRange converts Flutter date-only filters ("YYYY-MM-DD", inclusive on both ends,
// interpreted in the shop timezone) into a UTC half-open range [from, to).
// Full RFC3339 timestamps are also accepted. Empty values mean unbounded.
func DayRange(from, to string, loc *time.Location) (*time.Time, *time.Time, error) {
	var fromT, toT *time.Time
	if f := strings.TrimSpace(from); f != "" {
		t, dateOnly, err := parseDay(f, loc)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid from: %w", err)
		}
		_ = dateOnly
		u := t.UTC()
		fromT = &u
	}
	if s := strings.TrimSpace(to); s != "" {
		t, dateOnly, err := parseDay(s, loc)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid to: %w", err)
		}
		if dateOnly {
			t = t.AddDate(0, 0, 1) // include the whole "to" day
		}
		u := t.UTC()
		toT = &u
	}
	return fromT, toT, nil
}

func parseDay(s string, loc *time.Location) (time.Time, bool, error) {
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		return t, true, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, false, nil
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, loc); err == nil {
		return t, false, nil
	}
	return time.Time{}, false, fmt.Errorf("%w: unsupported date %q", ErrInvalidDate, s)
}

// TZOffset returns the MySQL CONVERT_TZ offset (e.g. "+07:00") of loc at the given time.
func TZOffset(loc *time.Location, at time.Time) string {
	_, sec := at.In(loc).Zone()
	sign := "+"
	if sec < 0 {
		sign = "-"
		sec = -sec
	}
	return fmt.Sprintf("%s%02d:%02d", sign, sec/3600, (sec%3600)/60)
}

// ISOTime formats a timestamp for Flutter in the shop timezone (RFC3339 with offset).
func ISOTime(t time.Time, loc *time.Location) string {
	return t.In(loc).Format(time.RFC3339)
}
