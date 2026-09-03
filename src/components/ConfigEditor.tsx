import React, { ChangeEvent } from 'react';
import { InlineField, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { CalendarSource, ColorRule, MyDataSourceOptions, MySecureJsonData } from '../types';
import { ColorRulesEditor } from './ColorRulesEditor';
import { additionalIcsUrlKey, CalendarsEditor } from './CalendarsEditor';

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions, MySecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData, secureJsonFields, secureJsonData } = options;
  const colorRules = jsonData.colorRules ?? [];
  const calendars = jsonData.calendars ?? [];

  // Secure field (only ever sent to / read by the backend, never the browser)
  const onIcsUrlChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        icsUrl: event.target.value,
      },
    });
  };

  const onResetIcsUrl = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        icsUrl: false,
      },
      secureJsonData: {
        ...secureJsonData,
        icsUrl: '',
      },
    });
  };

  const onColorRulesChange = (rules: ColorRule[]) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        colorRules: rules,
      },
    });
  };

  const onCalendarsChange = (nextCalendars: CalendarSource[]) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        calendars: nextCalendars,
      },
    });
  };

  const onAdditionalUrlChange = (id: string, value: string) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        [additionalIcsUrlKey(id)]: value,
      },
    });
  };

  const onAdditionalUrlReset = (id: string) => {
    const key = additionalIcsUrlKey(id);
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        [key]: false,
      },
      secureJsonData: {
        ...secureJsonData,
        [key]: '',
      },
    });
  };

  // Removes the calendar entry and clears its secret URL in one combined
  // update — doing this as two separate onOptionsChange calls would have the
  // second call overwrite the first's (still-stale) options snapshot.
  const onCalendarRemove = (id: string) => {
    const key = additionalIcsUrlKey(id);
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        calendars: calendars.filter((cal) => cal.id !== id),
      },
      secureJsonFields: {
        ...options.secureJsonFields,
        [key]: false,
      },
      secureJsonData: {
        ...secureJsonData,
        [key]: '',
      },
    });
  };

  return (
    <>
      <InlineField
        label="ICS-URL"
        labelWidth={14}
        interactive
        tooltip={
          "Google Kalender -> Einstellungen -> 'Kalender integrieren' -> öffentliche Adresse im iCal-Format (endet meist auf /basic.ics). Wird verschlüsselt gespeichert und nur serverseitig abgerufen."
        }
      >
        <SecretInput
          required
          id="config-editor-ics-url"
          isConfigured={secureJsonFields.icsUrl}
          value={secureJsonData?.icsUrl}
          placeholder="https://calendar.google.com/calendar/ical/xxxxx/public/basic.ics"
          width={60}
          onReset={onResetIcsUrl}
          onChange={onIcsUrlChange}
        />
      </InlineField>

      <div style={{ marginTop: '24px' }}>
        <h4>Weitere Kalender</h4>
        <p style={{ maxWidth: '640px' }}>
          Optional: zusätzliche ICS-Kalender, deren Termine mit denen des Hauptkalenders (oben) zusammengeführt
          werden. Farbregeln (siehe unten) gelten dann automatisch über alle Kalender hinweg.
        </p>
        <CalendarsEditor
          calendars={calendars}
          secureJsonData={secureJsonData ?? {}}
          secureJsonFields={secureJsonFields ?? {}}
          onCalendarsChange={onCalendarsChange}
          onUrlChange={onAdditionalUrlChange}
          onUrlReset={onAdditionalUrlReset}
          onCalendarRemove={onCalendarRemove}
        />
      </div>

      <div style={{ marginTop: '24px' }}>
        <h4>Farbregeln</h4>
        <p style={{ maxWidth: '640px' }}>
          Ordnet Textausschnitten aus dem Termin-Titel einen Zahlenwert zu, der im Feld <code>color_value</code>{' '}
          zurückgegeben wird. Reihenfolge = Priorität, die erste passende Regel gewinnt. Der Wert muss zu einer im
          Panel vorbereiteten Threshold-Stufe passen (siehe GitHub-README, Abschnitt &quot;Color rules&quot;).
        </p>
        <ColorRulesEditor rules={colorRules} onChange={onColorRulesChange} />
      </div>
    </>
  );
}
