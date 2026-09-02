package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

// Datasource fetches a Google Calendar ICS feed server-side (avoiding the
// browser CORS restriction Google's ICS export enforces) and serves the
// parsed events to Grafana.
type Datasource struct {
	icsURL     string
	httpClient *http.Client
}

// NewDatasource creates a new datasource instance.
func NewDatasource(_ context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	config, err := models.LoadPluginSettings(settings)
	if err != nil {
		return nil, err
	}

	icsURL := ""
	if config.Secrets != nil {
		icsURL = config.Secrets.IcsUrl
	}

	return &Datasource{
		icsURL:     icsURL,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewDatasource factory function.
func (d *Datasource) Dispose() {}

func (d *Datasource) fetchEvents(ctx context.Context, from, to time.Time) ([]CalendarEvent, error) {
	if d.icsURL == "" {
		return nil, errors.New("keine ICS-URL konfiguriert (Data Source Einstellungen -> ICS-URL)")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.icsURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ICS-Kalender konnte nicht abgerufen werden: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("ICS-URL antwortete mit HTTP %d: %s", resp.StatusCode, string(body))
	}

	events, err := parseICS(resp.Body, from, to)
	if err != nil {
		return nil, fmt.Errorf("ICS-Kalender konnte nicht geparst werden: %w", err)
	}
	return events, nil
}

// QueryData handles multiple queries and returns multiple responses.
// All queries in a panel share the same calendar and are answered from a
// single ICS fetch, using the widest time range requested across them.
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

	events, fetchErr := d.fetchEvents(ctx, from, to)

	for _, q := range req.Queries {
		if fetchErr != nil {
			response.Responses[q.RefID] = backend.ErrorResponseWithErrorSource(fetchErr)
			continue
		}
		response.Responses[q.RefID] = eventsToDataResponse(q, events)
	}
	return response, nil
}

type queryModel struct {
	MaxEvents float64 `json:"maxEvents"`
}

func eventsToDataResponse(query backend.DataQuery, events []CalendarEvent) backend.DataResponse {
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

	for i, e := range filtered {
		starts[i] = e.Start
		ends[i] = e.End
		titles[i] = e.Summary
		locations[i] = e.Location
		descriptions[i] = e.Description
		allDay[i] = e.AllDay
		uids[i] = e.UID
	}

	frame := data.NewFrame("events",
		data.NewField("time", nil, starts),
		data.NewField("end_time", nil, ends),
		data.NewField("title", nil, titles),
		data.NewField("location", nil, locations),
		data.NewField("description", nil, descriptions),
		data.NewField("all_day", nil, allDay),
		data.NewField("uid", nil, uids),
	)

	return backend.DataResponse{Frames: data.Frames{frame}}
}

// CheckHealth handles health checks sent from Grafana to the plugin, used by
// the "Save & Test" button on the datasource configuration page.
func (d *Datasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	now := time.Now()
	events, err := d.fetchEvents(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 1, 0))
	if err != nil {
		return &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: err.Error()}, nil
	}
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: fmt.Sprintf("Kalender erfolgreich geladen (%d Termine im nächsten Monat gefunden)", len(events)),
	}, nil
}
