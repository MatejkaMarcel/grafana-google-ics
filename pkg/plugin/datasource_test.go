package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"

	"github.com/custom/google-calendar-ics/pkg/models"
)

func TestQueryData(t *testing.T) {
	ds := Datasource{}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A"},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	if len(resp.Responses) != 1 {
		t.Fatal("QueryData must return a response")
	}
}

func TestColorValueForTitle(t *testing.T) {
	rules := []models.ColorRule{
		{Pattern: "Beispielmuster A", Value: 10},
		{Pattern: "Beispiel", Value: 20},
	}

	tests := []struct {
		name  string
		title string
		rules []models.ColorRule
		want  float64
	}{
		{"matches the more specific first rule", "Termin: Beispielmuster A", rules, 10},
		{"falls through to a later, broader rule", "Beispielmuster B", rules, 20},
		{"no matching rule", "Etwas ganz anderes", rules, 0},
		{"case-insensitive match", "TERMIN: BEISPIELMUSTER A", rules, 10},
		{"no rules configured", "Beispielmuster A", nil, 0},
		{"empty pattern is skipped", "Beispielmuster A", []models.ColorRule{{Pattern: "", Value: 99}}, 0},
		{
			"first matching rule wins even if a later rule would also match",
			"AB-Termin",
			[]models.ColorRule{{Pattern: "A", Value: 1}, {Pattern: "AB", Value: 2}},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := colorValueForTitle(tt.title, tt.rules); got != tt.want {
				t.Errorf("colorValueForTitle(%q) = %v, want %v", tt.title, got, tt.want)
			}
		})
	}
}

func icsHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}
}

const icsWithOneEvent = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:%s@example.com
DTSTAMP:20260101T000000Z
DTSTART:%sT090000Z
DTEND:%sT100000Z
SUMMARY:%s
END:VEVENT
END:VCALENDAR
`

func TestFetchEventsMergesAndSortsMultipleCalendars(t *testing.T) {
	serverA := httptest.NewServer(icsHandler(fmt.Sprintf(icsWithOneEvent, "a1", "20260115", "20260115", "Termin A")))
	defer serverA.Close()
	serverB := httptest.NewServer(icsHandler(fmt.Sprintf(icsWithOneEvent, "b1", "20260110", "20260110", "Termin B")))
	defer serverB.Close()

	ds := &Datasource{
		calendars: []calendarSource{
			{URL: serverA.URL},                     // primary, no name
			{Name: "Team B", URL: serverB.URL},      // additional, named
		},
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	events, warnings, err := ds.fetchEvents(context.Background(), from, to)
	if err != nil {
		t.Fatalf("fetchEvents returned an error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 merged events, got %d", len(events))
	}
	// Sorted by start: "Termin B" (10th) comes before "Termin A" (15th).
	if events[0].Summary != "Termin B" || events[0].CalendarName != "Team B" {
		t.Errorf("events[0] = %q from %q, want \"Termin B\" from \"Team B\"", events[0].Summary, events[0].CalendarName)
	}
	if events[1].Summary != "Termin A" || events[1].CalendarName != "" {
		t.Errorf("events[1] = %q from %q, want \"Termin A\" from the primary calendar (empty name)", events[1].Summary, events[1].CalendarName)
	}
}

func TestFetchEventsPartialFailureReturnsWarningNotError(t *testing.T) {
	serverA := httptest.NewServer(icsHandler(fmt.Sprintf(icsWithOneEvent, "a1", "20260115", "20260115", "Termin A")))
	defer serverA.Close()
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer serverB.Close()

	ds := &Datasource{
		calendars: []calendarSource{
			{URL: serverA.URL},
			{Name: "Kaputter Kalender", URL: serverB.URL},
		},
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	events, warnings, err := ds.fetchEvents(context.Background(), from, to)
	if err != nil {
		t.Fatalf("fetchEvents should not fail as long as at least one calendar succeeds: %v", err)
	}
	if len(events) != 1 || events[0].Summary != "Termin A" {
		t.Fatalf("expected the 1 event from the working calendar, got %+v", events)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Kaputter Kalender") {
		t.Errorf("expected a warning naming the failed calendar, got %v", warnings)
	}
}

func TestFetchEventsAllCalendarsFail(t *testing.T) {
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer serverA.Close()

	ds := &Datasource{
		calendars:  []calendarSource{{URL: serverA.URL}},
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	_, _, err := ds.fetchEvents(context.Background(), time.Now(), time.Now().AddDate(0, 1, 0))
	if err == nil {
		t.Fatal("expected an error when every configured calendar fails")
	}
}
