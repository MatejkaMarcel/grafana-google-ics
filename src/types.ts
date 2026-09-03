import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface MyQuery extends DataQuery {
  /** 0 or undefined = unlimited */
  maxEvents?: number;
}

export const DEFAULT_QUERY: Partial<MyQuery> = {};

/**
 * Maps a case-insensitive substring of an event title to a numeric value.
 * The backend writes the value of the first matching rule (list order =
 * priority) into the response's color_value field, defaulting to 0.
 */
export interface ColorRule {
  pattern: string;
  value: number;
}

/**
 * One additional (non-primary) calendar. `id` is generated once client-side
 * and stays stable across edits: it's used as the suffix of the matching
 * secureJsonData key ("icsUrl__<id>") so the URL keeps referring to the
 * same entry even after other calendars are added/removed/reordered.
 */
export interface CalendarSource {
  id: string;
  name: string;
}

/**
 * These are options configured for each DataSource instance
 */
export interface MyDataSourceOptions extends DataSourceJsonData {
  colorRules?: ColorRule[];
  calendars?: CalendarSource[];
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData {
  /** Primary calendar's ICS URL (kept for backward compatibility). */
  icsUrl?: string;
  /** Additional calendars' ICS URLs, keyed as "icsUrl__<CalendarSource.id>". */
  [key: string]: string | undefined;
}
