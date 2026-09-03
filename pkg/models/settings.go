package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// additionalIcsURLSecretPrefix namespaces the secureJsonData keys used for
// additional calendars' URLs, e.g. "icsUrl__<CalendarSource.ID>". The
// primary calendar keeps using the plain "icsUrl" key for backward
// compatibility with configurations created before multi-calendar support.
const additionalIcsURLSecretPrefix = "icsUrl__"

type PluginSettings struct {
	// Calendars holds the additional (non-primary) calendars: their display
	// name and a stable ID used to look up the matching URL secret. The
	// primary calendar's URL alone is stored under Secrets.IcsUrl.
	Calendars  []CalendarSource      `json:"calendars"`
	ColorRules []ColorRule           `json:"colorRules"`
	Secrets    *SecretPluginSettings `json:"-"`
}

// CalendarSource is one additional calendar configured on the data source.
// ID is generated client-side once and stays stable across edits so its
// URL secret (secureJsonData["icsUrl__"+ID]) keeps referring to the same
// entry even after other calendars are added or removed.
type CalendarSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ColorRule maps a case-insensitive substring of an event title to a
// numeric value, used to populate the color_value response field for
// panels (e.g. Business Calendar) that color events via thresholds.
type ColorRule struct {
	Pattern string  `json:"pattern"`
	Value   float64 `json:"value"`
}

type SecretPluginSettings struct {
	// IcsUrl is the primary calendar's URL (single, backward-compatible field).
	IcsUrl string
	// AdditionalIcsUrls maps a CalendarSource.ID to its URL.
	AdditionalIcsUrls map[string]string
}

func LoadPluginSettings(source backend.DataSourceInstanceSettings) (*PluginSettings, error) {
	settings := PluginSettings{}
	err := json.Unmarshal(source.JSONData, &settings)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
	}

	settings.Secrets = loadSecretPluginSettings(source.DecryptedSecureJSONData)

	return &settings, nil
}

func loadSecretPluginSettings(source map[string]string) *SecretPluginSettings {
	secrets := &SecretPluginSettings{
		IcsUrl:            source["icsUrl"],
		AdditionalIcsUrls: map[string]string{},
	}
	for key, value := range source {
		if id, ok := strings.CutPrefix(key, additionalIcsURLSecretPrefix); ok {
			secrets.AdditionalIcsUrls[id] = value
		}
	}
	return secrets
}
