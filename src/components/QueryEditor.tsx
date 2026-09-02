import React, { ChangeEvent } from 'react';
import { InlineField, Input, Stack } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { MyDataSourceOptions, MyQuery } from '../types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const onMaxEventsChange = (event: ChangeEvent<HTMLInputElement>) => {
    const raw = event.target.value;
    onChange({ ...query, maxEvents: raw === '' ? undefined : Number(raw) });
  };

  return (
    <Stack gap={0}>
      <InlineField
        label="Max. Termine"
        labelWidth={16}
        tooltip="0 oder leer = unbegrenzt (alle Termine im Dashboard-Zeitraum)"
      >
        <Input
          id="query-editor-max-events"
          type="number"
          min={0}
          width={12}
          value={query.maxEvents ?? ''}
          onChange={onMaxEventsChange}
          onBlur={onRunQuery}
        />
      </InlineField>
    </Stack>
  );
}
