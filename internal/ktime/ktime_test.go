package ktime

import (
	"testing"
	"time"
)

func TestScanStoredFormatWithOffset(t *testing.T) {
	// Form present in the production DB (UTC offset, 6-digit microseconds).
	const stored = "2026-01-03 13:21:44.873677+00:00"
	var ts Time
	if err := ts.Scan(stored); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if ts.Year() != 2026 || ts.Month() != time.January || ts.Day() != 3 {
		t.Fatalf("date = %v", ts.Time)
	}
	if ts.Nanosecond() != 873677000 {
		t.Fatalf("nanosecond = %d, want 873677000", ts.Nanosecond())
	}
	_, offset := ts.Zone()
	if offset != 0 {
		t.Fatalf("offset = %d, want 0 (+00:00)", offset)
	}
}

func TestScanKSTOffset(t *testing.T) {
	const stored = "2026-01-03 13:21:44.873677+09:00"
	var ts Time
	if err := ts.Scan(stored); err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, offset := ts.Zone()
	if offset != 9*60*60 {
		t.Fatalf("offset = %d, want 32400 (+09:00)", offset)
	}
}

func TestScanNoMicroseconds(t *testing.T) {
	var ts Time
	if err := ts.Scan("2026-01-03 13:21:44+00:00"); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if ts.Second() != 44 {
		t.Fatalf("second = %d", ts.Second())
	}
}

func TestValueEmitsPythonFormat(t *testing.T) {
	// A KST timestamp must serialize like Python's str(datetime.now(KST)).
	ts := New(time.Date(2026, 1, 3, 13, 21, 44, 873677000, KST))
	v, err := ts.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	const want = "2026-01-03 13:21:44.873677+09:00"
	if s, ok := v.(string); !ok || s != want {
		t.Fatalf("value = %v (%T), want %q", v, v, want)
	}
}

func TestRoundTrip(t *testing.T) {
	ts := New(time.Date(2026, 5, 30, 9, 18, 0, 123456000, KST))
	v, err := ts.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	var back Time
	if err := back.Scan(v.(string)); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !back.Time.Equal(ts.Time) {
		t.Fatalf("round trip = %v, want %v", back.Time, ts.Time)
	}
}

func TestNowKSTLocation(t *testing.T) {
	now := NowKST()
	if now.Location().String() != KST.String() {
		t.Fatalf("location = %s, want %s", now.Location(), KST)
	}
}

func TestScanNilZero(t *testing.T) {
	var ts Time
	if err := ts.Scan(nil); err != nil {
		t.Fatalf("scan nil: %v", err)
	}
	v, err := ts.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if v != nil {
		t.Fatalf("zero time value = %v, want nil", v)
	}
}

func TestNow(t *testing.T) {
	before := time.Now()
	got := Now()
	after := time.Now()
	if got.Time.Before(before) || got.Time.After(after) {
		t.Errorf("Now() = %v, not in [%v, %v]", got.Time, before, after)
	}
}

func TestKtimeString(t *testing.T) {
	ts := New(time.Date(2026, 1, 3, 13, 21, 44, 873677000, KST))
	s := ts.String()
	if s == "" {
		t.Error("String() returned empty for non-zero time")
	}
	var zero Time
	if zero.String() != "" {
		t.Errorf("zero String() = %q, want empty", zero.String())
	}
}

func TestScanTimeTime(t *testing.T) {
	var ts Time
	tt := time.Date(2026, 1, 3, 13, 21, 44, 0, KST)
	if err := ts.Scan(tt); err != nil {
		t.Fatalf("Scan time.Time: %v", err)
	}
	if ts.Year() != 2026 {
		t.Errorf("year = %d, want 2026", ts.Year())
	}
}

func TestScanUnknownType(t *testing.T) {
	var ts Time
	if err := ts.Scan(42); err == nil {
		t.Error("Scan int expected error, got nil")
	}
}

func TestParseInvalidString(t *testing.T) {
	var ts Time
	if err := ts.Scan("not-a-timestamp"); err == nil {
		t.Error("Scan invalid string expected error, got nil")
	}
}

func TestScanBytes(t *testing.T) {
	var ts Time
	if err := ts.Scan([]byte("2026-01-03 13:21:44+09:00")); err != nil {
		t.Fatalf("Scan []byte: %v", err)
	}
	if ts.Year() != 2026 {
		t.Errorf("year = %d, want 2026", ts.Year())
	}
}
