package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/custom/google-calendar-ics/pkg/models"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// calendarSource is one resolved (name + URL) calendar to fetch. Name is
// empty for the primary calendar.
type calendarSource struct {
	Name string
	URL  string
}

// Datasource fetches one or more Google Calendar ICS feeds server-side
// (avoiding the browser CORS restriction Google's ICS export enforces) and
// serves the merged, parsed events to Grafana.
type Datasource struct {
	calendars  []calendarSource
	colorRules []models.ColorRule
	httpClient *http.Client
}

// NewDatasource creates a new datasource instance.
func NewDatasource(_ context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	config, err := models.LoadPluginSettings(settings)
	if err != nil {
		return nil, err
	}

	var calendars []calendarSource
	if config.Secrets != nil {
		if config.Secrets.IcsUrl != "" {
			calendars = append(calendars, calendarSource{URL: config.Secrets.IcsUrl})
		}
		for _, c := range config.Calendars {
			url := config.Secrets.AdditionalIcsUrls[c.ID]
			if url == "" {
				continue
			}
			calendars = append(calendars, calendarSource{Name: c.Name, URL: url})
		}
	}

	return &Datasource{
		calendars:  calendars,
		colorRules: config.ColorRules,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewDatasource factory function.
func (d *Datasource) Dispose() {}

// fetchEvents fetches all configured calendars concurrently and merges the
// results. A calendar that fails to fetch or parse doesn't fail the whole
// query as long as at least one other calendar succeeds -- its error is
// returned as a warning instead.
func (d *Datasource) fetchEvents(ctx context.Context, from, to time.Time) (events []CalendarEvent, warnings []string, err error) {
	if len(d.calendars) == 0 {
		return nil, nil, errors.New("keine ICS-URL konfiguriert (Data Source Einstellungen -> ICS-URL)")
	}

	type result struct {
		name   string
		events []CalendarEvent
		err    error
	}
	results := make([]result, len(d.calendars))

	var wg sync.WaitGroup
	for i, cal := range d.calendars {
		wg.Add(1)
		go func(i int, cal calendarSource) {
			defer wg.Done()
			calEvents, calErr := d.fetchOneCalendar(ctx, cal, from, to)
			results[i] = result{name: cal.Name, events: calEvents, err: calErr}
		}(i, cal)
	}
	wg.Wait()

	failures := 0
	for _, r := range results {
		if r.err != nil {
			failures++
			label := r.name
			if label == "" {
				label = "Hauptkalender"
			}
			warnings = append(warnings, fmt.Sprintf("%s: %s", label, r.err.Error()))
			continue
		}
		events = append(events, r.events...)
	}

	if failures == len(d.calendars) {
		return nil, nil, fmt.Errorf("keiner der konfigurierten Kalender konnte geladen werden: %s", strings.Join(warnings, "; "))
	}

	sort.Slice(events, func(i, j int) bool { return events[i].Start.Before(events[j].Start) })
	return events, warnings, nil
}

func (d *Datasource) fetchOneCalendar(ctx context.Context, cal calendarSource, from, to time.Time) ([]CalendarEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cal.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ICS-Kalender konnte nicht abgerufen werden: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("ICS-URL antwortete mit HTTP %d: %s", resp.StatusCode, string(body))
	}

	events, err := parseICS(resp.Body, from, to)
	if err != nil {
		return nil, fmt.Errorf("ICS-Kalender konnte nicht geparst werden: %w", err)
	}
	for i := range events {
		events[i].CalendarName = cal.Name
	}
	return events, nil
}

// QueryData handles multiple queries and returns multiple responses.
// All queries in a panel share the same calendars and are answered from a
// single fetch, using the widest time range requested across them.
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	if len(req.Queries) == 0 {
		return response, nil
	}

	from, to := req.Queries[0].TimeRange.From, req.Queries[0].TimeRange.To
	for _, q := range req.Queries[1:] {
		if q.TimeRange.From.Before(from) {
			from = q.TimeRange.From
		}
		if q.TimeRange.To.After(to) {
			to = q.TimeRange.To
		}
	}

	events, warnings, fetchErr := d.fetchEvents(ctx, from, to)

	for _, q := range req.Queries {
		if fetchErr != nil {
			response.Responses[q.RefID] = backend.ErrorResponseWithErrorSource(fetchErr)
			continue
		}
		response.Responses[q.RefID] = d.eventsToDataResponse(q, events, warnings)
	}
	return response, nil
}

type queryModel struct {
	MaxEvents float64 `json:"maxEvents"`
}

func (d *Datasource) eventsToDataResponse(query backend.DataQuery, events []CalendarEvent, warnings []string) backend.DataResponse {
	var qm queryModel
	_ = json.Unmarshal(query.JSON, &qm)

	filtered := make([]CalendarEvent, 0, len(events))
	for _, e := range events {
		if e.Start.Before(query.TimeRange.To) && e.End.After(query.TimeRange.From) {
			filtered = append(filtered, e)
		}
	}
	if qm.MaxEvents > 0 && int(qm.MaxEvents) < len(filtered) {
		filtered = filtered[:int(qm.MaxEvents)]
	}

	starts := make([]time.Time, len(filtered))
	ends := make([]time.Time, len(filtered))
	titles := make([]string, len(filtered))
	locations := make([]string, len(filtered))
	descriptions := make([]string, len(filtered))
	allDay := make([]bool, len(filtered))
	uids := make([]string, len(filtered))
	colorValues := make([]float64, len(filtered))
	calendarNames := make([]string, len(filtered))

	for i, e := range filtered {
		starts[i] = e.Start
		ends[i] = e.End
		titles[i] = e.Summary
		locations[i] = e.Location
		descriptions[i] = e.Description
		allDay[i] = e.AllDay
		uids[i] = e.UID
		colorValues[i] = colorValueForTitle(e.Summary, d.colorRules)
		calendarNames[i] = e.CalendarName
	}

	frame := data.NewFrame("events",
		data.NewField("time", nil, starts),
		data.NewField("end_time", nil, ends),
		data.NewField("title", nil, titles),
		data.NewField("location", nil, locations),
		data.NewField("description", nil, descriptions),
		data.NewField("all_day", nil, allDay),
		data.NewField("uid", nil, uids),
		data.NewField("color_value", nil, colorValues),
		data.NewField("calendar", nil, calendarNames),
	)

	if len(warnings) > 0 {
		notices := make([]data.Notice, len(warnings))
		for i, w := range warnings {
			notices[i] = data.Notice{Severity: data.NoticeSeverityWarning, Text: w}
		}
		frame.Meta = &data.FrameMeta{Notices: notices}
	}

	return backend.DataResponse{Frames: data.Frames{frame}}
}

// colorValueForTitle returns the value of the first color rule whose
// pattern is a case-insensitive substring of title (list order = priority),
// or 0 if no rule matches (or none are configured).
func colorValueForTitle(title string, rules []models.ColorRule) float64 {
	lowerTitle := strings.ToLower(title)
	for _, rule := range rules {
		if rule.Pattern == "" {
			continue
		}
		if strings.Contains(lowerTitle, strings.ToLower(rule.Pattern)) {
			return rule.Value
		}
	}
	return 0
}

// CheckHealth handles health checks sent from Grafana to the plugin, used by
// the "Save & Test" button on the datasource configuration page.
func (d *Datasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	now := time.Now()
	events, warnings, err := d.fetchEvents(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 1, 0))
	if err != nil {
		return &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: err.Error()}, nil
	}

	message := fmt.Sprintf("%d Kalender erfolgreich geladen (%d Termine im nächsten Monat gefunden)", len(d.calendars)-len(warnings), len(events))
	if len(warnings) > 0 {
		message += fmt.Sprintf(" -- Achtung, fehlgeschlagen: %s", strings.Join(warnings, "; "))
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: message,
	}, nil
}
