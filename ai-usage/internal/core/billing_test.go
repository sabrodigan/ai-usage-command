package core

import (
	"testing"
	"time"
)

func TestCalculateBillingCycle_CalendarMonth(t *testing.T) {
	date := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	window := CalculateBillingCycle(date, 1)

	expectedStart := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

	if !window.Start.Equal(expectedStart) {
		t.Errorf("expected start %v, got %v", expectedStart, window.Start)
	}
	if !window.End.Equal(expectedEnd) {
		t.Errorf("expected end %v, got %v", expectedEnd, window.End)
	}
}

func TestCalculateBillingCycle_CustomAnchor(t *testing.T) {
	// Date is after anchor day (20th > 15th)
	dateAfter := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	w1 := CalculateBillingCycle(dateAfter, 15)

	expStart1 := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	expEnd1 := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

	if !w1.Start.Equal(expStart1) || !w1.End.Equal(expEnd1) {
		t.Errorf("case after anchor failed: expected %v..%v, got %v..%v", expStart1, expEnd1, w1.Start, w1.End)
	}

	// Date is before anchor day (10th < 15th)
	dateBefore := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	w2 := CalculateBillingCycle(dateBefore, 15)

	expStart2 := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	expEnd2 := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

	if !w2.Start.Equal(expStart2) || !w2.End.Equal(expEnd2) {
		t.Errorf("case before anchor failed: expected %v..%v, got %v..%v", expStart2, expEnd2, w2.Start, w2.End)
	}
}
