# Changelog

## 1.0.3

- New: configurable "Color rules" on the data source (title pattern →
  numeric value). The backend writes the value of the first matching rule
  into a new `color_value` field on every event (`0` if none match), for
  calendar panels that color events from numeric thresholds (e.g. Business
  Calendar). See the README's "Color rules" section.
- New: support for multiple calendars per data source. The existing ICS-URL
  field remains the primary calendar; a new repeatable "Weitere Kalender"
  (additional calendars) list lets you add more ICS URLs, each with its own
  display name. All configured calendars are fetched concurrently and their
  events merged into one combined, time-sorted result -- color rules apply
  across the merged set automatically.
- New `calendar` response field identifying which configured calendar an
  event came from (empty string for the primary calendar).
- A calendar that fails to fetch or parse no longer fails the whole query as
  long as at least one other configured calendar succeeds; the failure is
  surfaced as a panel notice/warning instead. "Save & Test" and the failure
  message name the calendar that failed.
- Fully backward compatible: existing single-calendar configurations keep
  working unchanged.

## 1.0.2

- Fix: all-day (whole-day) events could render across two days in calendar
  panels. DATE-only ICS values are now anchored to the calendar's own
  `X-WR-TIMEZONE` instead of UTC, so a viewer in a positive UTC-offset time
  zone (e.g. Europe/Berlin) no longer sees the event's end drift past
  midnight into the next day.

## 1.0.1

- Bump `@grafana/data`/`runtime`/`ui`/`schema`/`i18n` to 13.2.0 (requires React 19)
- Bump `@grafana/eslint-config` to 10.0.0, `@stylistic/eslint-plugin-ts` replaced by `@stylistic/eslint-plugin`
- Fix CI workflow (removed a leftover `npm run test:ci` step after Jest was dropped)

## 1.0.0

Initial release.
