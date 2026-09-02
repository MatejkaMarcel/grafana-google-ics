import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface MyQuery extends DataQuery {
  /** 0 or undefined = unlimited */
  maxEvents?: number;
}

export const DEFAULT_QUERY: Partial<MyQuery> = {};

/**
 * These are options configured for each DataSource instance
 */
export type MyDataSourceOptions = DataSourceJsonData;

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData {
  icsUrl?: string;
}
