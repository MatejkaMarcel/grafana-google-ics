package plugin

import (
	"strings"
	"testing"
	"time"
)

const sampleICS = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Google Inc//Google Calendar 70.9054//EN
X-WR-TIMEZONE:Europe/Berlin
BEGIN:VEVENT
UID:single-event@example.com
DTSTAMP:20260101T000000Z
DTSTART;TZID=Europe/Berlin:20260115T090000
DTEND;TZID=Europe/Berlin:20260115T100000
SUMMARY:Einzeltermin
LOCATION:Buero
END:VEVENT
BEGIN:VEVENT
UID:allday-event@example.com
DTSTAMP:20260101T000000Z
DTSTART;VALUE=DATE:20260120
DTEND;VALUE=DATE:20260121
SUMMARY:Ganztaegig
END:VEVENT
BEGIN:VEVENT
UID:weekly-series@example.com
DTSTAMP:20260101T000000Z
DTSTART;TZID=Europe/Berlin:20260106T140000
DTEND;TZID=Europe/Berlin:20260106T150000
SUMMARY:Wochenmeeting
RRULE:FREQ=WEEKLY;BYDAY=TU;COUNT=6
EXDATE;TZID=Europe/Berlin:20260120T140000
END:VEVENT
BEGIN:VEVENT
UID:weekly-series@example.com
RECURRENCE-ID;TZID=Europe/Berlin:20260127T140000
DTSTAMP:20260101T000000Z
DTSTART;TZID=Europe/Berlin:20260127T160000
DTEND;TZID=Europe/Berlin:20260127T170000
SUMMARY:Wochenmeeting (verschoben)
END:VEVENT
END:VCALENDAR
`

func TestParseICS(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	events, err := parseICS(strings.NewReader(sampleICS), from, to)
	if err != nil {
		t.Fatalf("parseICS returned an error: %v", err)
	}

	byUID := map[string][]CalendarEvent{}
	for _, e := range events {
		byUID[e.UID] = append(byUID[e.UID], e)
	}

	// 1. Simple single event with an explicit timezone.
	single := byUID["single-event@example.com"]
	if len(single) != 1 {
		t.Fatalf("expected 1 single event, got %d", len(single))
	}
	loc, _ := time.LoadLocation("Europe/Berlin")
	wantStart := time.Date(2026, 1, 15, 9, 0, 0, 0, loc)
	if !single[0].Start.Equal(wantStart) {
		t.Errorf("single event start = %v, want %v", single[0].Start, wantStart)
	}
	if single[0].Location != "Buero" {
		t.Errorf("single event location = %q, want %q", single[0].Location, "Buero")
	}

	// 2. All-day event: DTEND defaults correctly and AllDay is set.
	allday := byUID["allday-event@example.com"]
	if len(allday) != 1 {
		t.Fatalf("expected 1 all-day event, got %d", len(allday))
	}
	if !allday[0].AllDay {
		t.Errorf("all-day event should have AllDay=true")
	}
	if !allday[0].End.Equal(allday[0].Start.Add(24 * time.Hour)) {
		t.Errorf("all-day event end = %v, want start+24h", allday[0].End)
	}

	// 3. Weekly series: COUNT=6 minus 1 EXDATE minus 1 RECURRENCE-ID override
	// (which is re-emitted at its new time) => still 5 occurrences total in range,
	// one of them with the overridden title/time.
	series := byUID["weekly-series@example.com"]
	if len(series) != 5 {
		var starts []string
		for _, e := range series {
			starts = append(starts, e.Start.Format(time.RFC3339))
		}
		t.Fatalf("expected 5 occurrences (6 - 1 EXDATE), got %d: %v", len(series), starts)
	}

	var foundOverride bool
	for _, e := range series {
		if e.Summary == "Wochenmeeting (verschoben)" {
			foundOverride = true
			wantOverrideStart := time.Date(2026, 1, 27, 16, 0, 0, 0, loc)
			if !e.Start.Equal(wantOverrideStart) {
				t.Errorf("overridden occurrence start = %v, want %v", e.Start, wantOverrideStart)
			}
		}
		// The excluded occurrence (2026-01-20 14:00 Berlin) must not appear.
		excluded := time.Date(2026, 1, 20, 14, 0, 0, 0, loc)
		if e.Start.Equal(excluded) {
			t.Errorf("EXDATE occurrence at %v should have been excluded", excluded)
		}
	}
	if !foundOverride {
		t.Errorf("expected to find the RECURRENCE-ID override occurrence")
	}
}

func TestParseICSRangeFiltering(t *testing.T) {
	// Only ask for a range that excludes the single event and the all-day
	// event, but still overlaps part of the weekly series.
	from := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 29, 0, 0, 0, 0, time.UTC)

	events, err := parseICS(strings.NewReader(sampleICS), from, to)
	if err != nil {
		t.Fatalf("parseICS returned an error: %v", err)
	}
	for _, e := range events {
		if e.UID == "single-event@example.com" {
			t.Errorf("single event (2026-01-15) should not be in range %v..%v", from, to)
		}
	}
}

// TestParseICSAllDayAnchoredToCalendarTimezone reproduces the reported bug:
// a one-day all-day event was rendered across two days in the calendar panel
// because DATE-only values were anchored to UTC. Once the viewer's browser
// re-renders a UTC midnight in a positive-offset time zone (e.g. Europe/Berlin,
// UTC+2 in September), it drifts into the early morning of the same day and,
// for DTEND, past midnight of the *next* day -- which is exactly what made a
// single-day event appear to span two days. Anchoring to the calendar's own
// X-WR-TIMEZONE keeps the event exactly on its intended day.
func TestParseICSAllDayAnchoredToCalendarTimezone(t *testing.T) {
	const icsWithAllDayEvent = `BEGIN:VCALENDAR
VERSION:2.0
X-WR-TIMEZONE:Europe/Berlin
BEGIN:VEVENT
UID:one-day@example.com
DTSTAMP:20260101T000000Z
DTSTART;VALUE=DATE:20260929
DTEND;VALUE=DATE:20260930
SUMMARY:Ein Tag
END:VEVENT
END:VCALENDAR
`
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	events, err := parseICS(strings.NewReader(icsWithAllDayEvent), from, to)
	if err != nil {
		t.Fatalf("parseICS returned an error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("Europe/Berlin tzdata not available in this environment: %v", err)
	}

	wantStart := time.Date(2026, 9, 29, 0, 0, 0, 0, berlin)
	wantEnd := time.Date(2026, 9, 30, 0, 0, 0, 0, berlin)
	if !events[0].Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v (anchored to the calendar's own time zone, not UTC)", events[0].Start, wantStart)
	}
	if !events[0].End.Equal(wantEnd) {
		t.Errorf("End = %v, want %v", events[0].End, wantEnd)
	}
}
