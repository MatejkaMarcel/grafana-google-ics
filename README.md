# Google Calendar (ICS) Datasource for Grafana

![Grafana](https://img.shields.io/badge/Grafana-%3E%3D12.3.0-orange?logo=grafana&logoColor=white)
![Backend](https://img.shields.io/badge/backend-Go-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-Apache--2.0-blue)

A Grafana datasource plugin that turns a public **Google Calendar ICS/iCal feed**
into queryable panel data — tables, or a real calendar view via the
[Calendar panel](https://grafana.com/grafana/plugins/marcusolsson-calendar-panel/).

## Why a backend plugin?

Google's calendar export (`.../basic.ics`) does not send CORS headers, so a
purely frontend plugin that fetches it directly from the browser fails. This
plugin ships a small **Go backend** that fetches and parses the ICS feed
server-side. As a side effect, the calendar URL is stored as an encrypted
secret and is never sent to the browser.

## Features

- Fetches the ICS feed server-side (no CORS issues)
- Parses recurring events (`RRULE`), exceptions (`EXDATE`), and individually
  moved/cancelled occurrences (`RECURRENCE-ID` / `STATUS:CANCELLED`) — the
  pattern Google Calendar uses for edited recurring events
- ICS URL stored as an encrypted secret, never exposed to the browser
- Optional per-query result limit ("Max events")
- All-day events are anchored to the calendar's own time zone
  (`X-WR-TIMEZONE`), not UTC, so they render as exactly one day regardless
  of the viewer's time zone

## Installation

This plugin is unsigned (private/custom plugin, not published to the Grafana
catalog), so it's installed manually:

1. Build it (see [Development](#development) below) or grab a prebuilt
   `dist/` for your platform.
2. Copy the `dist/` contents to
   `<grafana-plugins-dir>/custom-googlecalendarics-datasource/` on your
   Grafana server.
3. Allow the unsigned plugin in `grafana.ini`:

   ```ini
   [plugins]
   allow_loading_unsigned_plugins = custom-googlecalendarics-datasource
   ```

4. Restart Grafana.

## Configuration

**Administration → Data Sources → Add data source → "Google Calendar ICS"**

| Field | Description |
|---|---|
| ICS-URL | Public iCal URL from Google Calendar (Settings → *Integrate calendar* → *Public address in iCal format*, usually ends in `/basic.ics`). Stored encrypted. |

Click **Save & Test** — the backend fetches the feed and reports how many
events it found in the next month.

## Query options

| Field | Description |
|---|---|
| Max. Termine (max events) | `0`/empty = unlimited. Caps the number of rows returned per query. |

## Fields returned

| Field | Type | Description |
|---|---|---|
| `time` | time | Event start |
| `end_time` | time | Event end |
| `title` | string | Summary |
| `location` | string | Location |
| `description` | string | Description |
| `all_day` | bool | `true` for all-day events |
| `uid` | string | Calendar event UID |

## Building a calendar view

Field names line up directly with the [Business Calendar
panel](https://grafana.com/grafana/plugins/marcusolsson-calendar-panel/)
(`marcusolsson-calendar-panel`, formerly "Calendar" by Marcus Olsson, now
maintained by Volkov Labs): install it, then map `time` → *Time field*,
`end_time` → *End time field*, `title` → *Text field*, `location` →
*Location field*.

## Known limitations

- `EXRULE` (deprecated, essentially never produced by Google Calendar) is ignored
- `RDATE` (extra occurrences added without their own `RRULE`) is not evaluated
- Timezones are resolved via the `TZID` parameter against the host's IANA
  timezone database; falls back to UTC if resolution fails
- All-day events are anchored to the calendar's `X-WR-TIMEZONE`; if a feed
  doesn't set that property, all-day events fall back to UTC (which can
  reintroduce the two-day rendering issue for viewers in a non-UTC time zone)

## Development

Requires Node.js ≥18 and Go ≥1.21, plus [Mage](https://magefile.org/).

```bash
# Frontend
npm install
npm run dev     # watch mode
npm run build   # production build

# Backend
mage -v build:linux   # or build:windows / build:darwin / buildAll

# Tests
npm run typecheck
npm run lint
go test ./...
```

Webpack, tsconfig, and the Mage build targets under `.config/` are managed by
`@grafana/create-plugin` — don't hand-edit `.config/`.

## License

Apache-2.0, see [LICENSE](LICENSE).
