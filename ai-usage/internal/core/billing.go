package core

import "time"

// CalculateBillingCycle determines the current cycle start and end boundaries.
// anchorDay: The day of the month the billing cycle resets on (1 = 1st of month to end of month).
func CalculateBillingCycle(now time.Time, anchorDay int) BillingWindow {
	if anchorDay < 1 || anchorDay > 28 {
		anchorDay = 1
	}

	year, month, day := now.Date()
	location := now.Location()

	var start, end time.Time

	if anchorDay == 1 {
		// Calendar month
		start = time.Date(year, month, 1, 0, 0, 0, 0, location)
		// Last moment of the month
		end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	} else {
		// Custom anchor day
		if day >= anchorDay {
			start = time.Date(year, month, anchorDay, 0, 0, 0, 0, location)
			end = time.Date(year, month, anchorDay, 0, 0, 0, 0, location).AddDate(0, 1, 0).Add(-time.Nanosecond)
		} else {
			start = time.Date(year, month, anchorDay, 0, 0, 0, 0, location).AddDate(0, -1, 0)
			end = time.Date(year, month, anchorDay, 0, 0, 0, 0, location).Add(-time.Nanosecond)
		}
	}

	return BillingWindow{
		Start: start,
		End:   end,
	}
}
