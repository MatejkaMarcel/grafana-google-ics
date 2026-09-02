import React, { ChangeEvent } from 'react';
import { InlineField, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions, MySecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { secureJsonFields, secureJsonData } = options;

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
    </>
  );
}
