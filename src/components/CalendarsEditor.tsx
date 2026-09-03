import React, { ChangeEvent } from 'react';
import { Button, IconButton, InlineField, InlineFieldRow, Input, SecretInput } from '@grafana/ui';
import { CalendarSource } from '../types';

export function additionalIcsUrlKey(id: string): string {
  return `icsUrl__${id}`;
}

function generateId(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

interface Props {
  calendars: CalendarSource[];
  secureJsonData: Record<string, string | undefined>;
  secureJsonFields: Record<string, boolean | undefined>;
  onCalendarsChange: (calendars: CalendarSource[]) => void;
  onUrlChange: (id: string, value: string) => void;
  onUrlReset: (id: string) => void;
  onCalendarRemove: (id: string) => void;
}

export function CalendarsEditor({
  calendars,
  secureJsonData,
  secureJsonFields,
  onCalendarsChange,
  onUrlChange,
  onUrlReset,
  onCalendarRemove,
}: Props) {
  const updateName = (index: number, name: string) => {
    onCalendarsChange(calendars.map((cal, i) => (i === index ? { ...cal, name } : cal)));
  };

  const addCalendar = () => {
    onCalendarsChange([...calendars, { id: generateId(), name: '' }]);
  };

  return (
    <div>
      {calendars.map((cal, index) => {
        const urlKey = additionalIcsUrlKey(cal.id);
        return (
          <InlineFieldRow key={cal.id}>
            <InlineField label="Name" labelWidth={10} tooltip="Frei wählbarer Anzeigename, z.B. 'Team A'">
              <Input
                width={20}
                placeholder="z.B. Team A"
                value={cal.name}
                onChange={(e: ChangeEvent<HTMLInputElement>) => updateName(index, e.currentTarget.value)}
              />
            </InlineField>
            <InlineField label="ICS-URL" labelWidth={12} tooltip="Öffentliche iCal-Adresse dieses zusätzlichen Kalenders">
              <SecretInput
                width={40}
                isConfigured={Boolean(secureJsonFields[urlKey])}
                value={secureJsonData[urlKey] ?? ''}
                placeholder="https://calendar.google.com/calendar/ical/xxxxx/public/basic.ics"
                onChange={(e: ChangeEvent<HTMLInputElement>) => onUrlChange(cal.id, e.currentTarget.value)}
                onReset={() => onUrlReset(cal.id)}
              />
            </InlineField>
            <IconButton
              name="trash-alt"
              aria-label="Kalender entfernen"
              tooltip="Kalender entfernen"
              onClick={() => onCalendarRemove(cal.id)}
            />
          </InlineFieldRow>
        );
      })}
      <Button icon="plus" variant="secondary" size="sm" onClick={addCalendar}>
        Kalender hinzufügen
      </Button>
    </div>
  );
}
