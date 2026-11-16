package adapter

import (
	"testing"
	"time"
)

func TestDayRangeUsesShopTimezoneAndIncludesWholeToDay(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	from, to, err := DayRange("2026-11-01", "2026-11-07", loc)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := from.Format(time.RFC3339), "2026-10-31T17:00:00Z"; got != want {
		t.Errorf("from = %s, want %s", got, want)
	}
	if got, want := to.Format(time.RFC3339), "2026-11-07T17:00:00Z"; got != want {
		t.Errorf("to = %s, want %s (exclusive end of 2026-11-07 local)", got, want)
	}
}

func TestDayRangeEmptyIsUnbounded(t *testing.T) {
	from, to, err := DayRange("", "", time.UTC)
	if err != nil || from != nil || to != nil {
		t.Fatalf("want nil,nil,nil got %v %v %v", from, to, err)
	}
}

func TestDayRangeRejectsGarbage(t *testing.T) {
	if _, _, err := DayRange("07/11/2026", "", time.UTC); err == nil {
		t.Fatal("expected error")
	}
}

func TestFloatStrRoundsSumArtifacts(t *testing.T) {
	if got := FloatStr(3454.450000000004); got != "3454.45" {
		t.Errorf("got %s", got)
	}
	if got := FloatStr(2); got != "2" {
		t.Errorf("got %s", got)
	}
}

func TestTZOffset(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	if got := TZOffset(loc, time.Now()); got != "+07:00" {
		t.Errorf("got %s", got)
	}
}
