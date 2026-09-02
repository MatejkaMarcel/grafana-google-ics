# Google Calendar ICS

Zeigt Termine aus einem öffentlichen Google-Kalender (ICS/iCal-Feed) in Grafana an — als Tabelle oder, in Kombination mit dem [Calendar-Panel](https://grafana.com/grafana/plugins/marcusolsson-calendar-panel/), als echte Monatsansicht.

## Funktionen

- Ruft den ICS-Feed **serverseitig** über das Plugin-Backend ab (keine CORS-Probleme wie bei einem reinen Browser-Abruf)
- Unterstützt wiederkehrende Termine (RRULE), Ausnahmen (EXDATE) und einzeln verschobene/abgesagte Termine (RECURRENCE-ID)
- Die ICS-URL wird verschlüsselt gespeichert und nie an den Browser übertragen
- Ganztägige Termine werden anhand der Kalender-eigenen Zeitzone (`X-WR-TIMEZONE`) statt UTC verankert, damit sie in Kalender-Panels als genau ein Tag erscheinen

## Voraussetzungen

- Eine öffentliche Google-Kalender-ICS-URL (Google Calendar → Einstellungen → "Kalender integrieren" → "Öffentliche Adresse im iCal-Format", endet auf `/basic.ics`)

## Erste Schritte

1. **Administration → Data Sources → Add data source → "Google Calendar ICS"**
2. Feld **ICS-URL** ausfüllen und **Save & Test**
3. Dashboard → Panel hinzufügen → diese Data Source auswählen
4. Als Tabelle: Felder `time`, `end_time`, `title`, `location` anzeigen
5. Als Kalender: [Calendar-Panel](https://grafana.com/grafana/plugins/marcusolsson-calendar-panel/) installieren und `time`/`end_time`/`title`/`location` als Field-Mapping eintragen

## Bereitgestellte Felder

| Feld | Beschreibung |
|---|---|
| `time` | Beginn des Termins |
| `end_time` | Ende des Termins |
| `title` | Titel/Zusammenfassung |
| `location` | Ort |
| `description` | Beschreibung |
| `all_day` | `true` bei ganztägigen Terminen |
| `uid` | Eindeutige Termin-ID aus dem Kalender |
