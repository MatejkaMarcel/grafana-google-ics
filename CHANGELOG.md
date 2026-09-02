# Changelog

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
