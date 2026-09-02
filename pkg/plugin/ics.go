package plugin

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// CalendarEvent is a single (possibly RRULE-expanded) calendar occurrence.
type CalendarEvent struct {
	UID         string
	Summary     string
	Location    string
	Description string
	Start       time.Time
	End         time.Time
	AllDay      bool
}

type icsProperty struct {
	name   string
	params map[string]string
	value  string
}

type rawEvent struct {
	props []icsProperty
}

// parseICS parses raw ICS bytes into a slice of expanded CalendarEvent
// occurrences that overlap [rangeFrom, rangeTo]. Recurring events (RRULE),
// excluded dates (EXDATE) and single-instance overrides (RECURRENCE-ID) are
// resolved. Some rarely used RFC5545 features (EXRULE, RDATE with PERIOD
// values, ...) are intentionally not supported.
func parseICS(r io.Reader, rangeFrom, rangeTo time.Time) ([]CalendarEvent, error) {
	lines, err := unfoldLines(r)
	if err != nil {
		return nil, err
	}

	var events []rawEvent
	var current *rawEvent

	for _, line := range lines {
		prop, ok := parseLine(line)
		if !ok {
			continue
		}
		switch {
		case prop.name == "BEGIN" && prop.value == "VEVENT":
			current = &rawEvent{}
		case prop.name == "END" && prop.value == "VEVENT":
			if current != nil {
				events = append(events, *current)
				current = nil
			}
		case current != nil:
			current.props = append(current.props, prop)
		}
	}

	type group struct {
		master    *rawEvent
		overrides []rawEvent
	}
	groups := map[string]*group{}
	var order []string

	for idx, ev := range events {
		uidProp, hasUID := getProp(ev, "UID")
		uid := uidProp.value
		if !hasUID || uid == "" {
			uid = fmt.Sprintf("__no-uid-%d", idx)
		}
		g, ok := groups[uid]
		if !ok {
			g = &group{}
			groups[uid] = g
			order = append(order, uid)
		}
		if _, hasRecID := getProp(ev, "RECURRENCE-ID"); hasRecID {
			g.overrides = append(g.overrides, ev)
		} else {
			evCopy := ev
			g.master = &evCopy
		}
	}

	var out []CalendarEvent

	for _, uid := range order {
		g := groups[uid]

		if g.master == nil {
			for _, ov := range g.overrides {
				if e, ok := eventFromProps(uid, ov.props); ok && overlaps(e.Start, e.End, rangeFrom, rangeTo) {
					out = append(out, e)
				}
			}
			continue
		}

		base, ok := eventFromProps(uid, g.master.props)
		if !ok {
			continue
		}

		rruleProp, hasRRule := getProp(*g.master, "RRULE")
		if !hasRRule {
			if overlaps(base.Start, base.End, rangeFrom, rangeTo) {
				out = append(out, base)
			}
			continue
		}

		duration := base.End.Sub(base.Start)

		ro, err := rrule.StrToROption(rruleProp.value)
		if err != nil {
			if overlaps(base.Start, base.End, rangeFrom, rangeTo) {
				out = append(out, base)
			}
			continue
		}
		ro.Dtstart = base.Start
		rr, err := rrule.NewRRule(*ro)
		if err != nil {
			if overlaps(base.Start, base.End, rangeFrom, rangeTo) {
				out = append(out, base)
			}
			continue
		}

		// Look a bit before rangeFrom too, so multi-day/long events that
		// started earlier but still run into the range are not missed.
		occurrences := rr.Between(rangeFrom.Add(-31*24*time.Hour), rangeTo, true)

		exdates := map[string]bool{}
		for _, p := range g.master.props {
			if p.name != "EXDATE" {
				continue
			}
			for _, v := range strings.Split(p.value, ",") {
				if t, _, err := parseICSTime(v, p.params); err == nil {
					exdates[occurrenceKey(t)] = true
				}
			}
		}

		overridesByOccurrence := map[string]CalendarEvent{}
		cancelled := map[string]bool{}
		for _, ov := range g.overrides {
			recIDProp, hasRecID := getProp(ov, "RECURRENCE-ID")
			if !hasRecID {
				continue
			}
			recID, _, err := parseICSTime(recIDProp.value, recIDProp.params)
			if err != nil {
				continue
			}
			key := occurrenceKey(recID)
			if statusProp, hasStatus := getProp(ov, "STATUS"); hasStatus && strings.EqualFold(statusProp.value, "CANCELLED") {
				cancelled[key] = true
				continue
			}
			if e, ok := eventFromProps(uid, ov.props); ok {
				overridesByOccurrence[key] = e
			}
		}

		seenOverrideKeys := map[string]bool{}
		for _, occ := range occurrences {
			key := occurrenceKey(occ)
			if exdates[key] || cancelled[key] {
				continue
			}
			if override, ok := overridesByOccurrence[key]; ok {
				seenOverrideKeys[key] = true
				if overlaps(override.Start, override.End, rangeFrom, rangeTo) {
					out = append(out, override)
				}
				continue
			}
			e := base
			e.Start = occ
			e.End = occ.Add(duration)
			if overlaps(e.Start, e.End, rangeFrom, rangeTo) {
				out = append(out, e)
			}
		}

		// Overrides that moved outside the originally generated occurrence
		// set (e.g. shifted far away) are still shown if in range.
		for key, override := range overridesByOccurrence {
			if !seenOverrideKeys[key] && overlaps(override.Start, override.End, rangeFrom, rangeTo) {
				out = append(out, override)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out, nil
}

func occurrenceKey(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func getProp(re rawEvent, name string) (icsProperty, bool) {
	for _, p := range re.props {
		if p.name == name {
			return p, true
		}
	}
	return icsProperty{}, false
}

func overlaps(start, end, rangeFrom, rangeTo time.Time) bool {
	if !end.After(start) {
		end = start.Add(time.Minute)
	}
	return start.Before(rangeTo) && end.After(rangeFrom)
}

func eventFromProps(uid string, props []icsProperty) (CalendarEvent, bool) {
	e := CalendarEvent{UID: uid}
	var haveStart, haveEnd bool
	var duration time.Duration

	for _, p := range props {
		switch p.name {
		case "SUMMARY":
			e.Summary = unescapeText(p.value)
		case "LOCATION":
			e.Location = unescapeText(p.value)
		case "DESCRIPTION":
			e.Description = unescapeText(p.value)
		case "DTSTART":
			if t, allDay, err := parseICSTime(p.value, p.params); err == nil {
				e.Start = t
				e.AllDay = allDay
				haveStart = true
			}
		case "DTEND":
			if t, _, err := parseICSTime(p.value, p.params); err == nil {
				e.End = t
				haveEnd = true
			}
		case "DURATION":
			if d, err := parseICSDuration(p.value); err == nil {
				duration = d
			}
		}
	}

	if !haveStart {
		return e, false
	}
	if !haveEnd {
		switch {
		case duration > 0:
			e.End = e.Start.Add(duration)
		case e.AllDay:
			e.End = e.Start.Add(24 * time.Hour)
		default:
			e.End = e.Start
		}
	}
	return e, true
}

func unfoldLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var lines []string
	for scanner.Scan() {
		raw := strings.TrimRight(scanner.Text(), "\r")
		if raw == "" {
			continue
		}
		if (raw[0] == ' ' || raw[0] == '\t') && len(lines) > 0 {
			lines[len(lines)-1] += raw[1:]
		} else {
			lines = append(lines, raw)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// parseLine splits a single unfolded ICS line "NAME;PARAM=VAL:VALUE" into
// name/params/value, respecting quoted param values that may contain ':'.
func parseLine(line string) (icsProperty, bool) {
	inQuotes := false
	sepIdx := -1
	for i, r := range line {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ':':
			if !inQuotes {
				sepIdx = i
			}
		}
		if sepIdx != -1 {
			break
		}
	}
	if sepIdx == -1 {
		return icsProperty{}, false
	}
	head := line[:sepIdx]
	value := line[sepIdx+1:]

	parts := strings.Split(head, ";")
	name := strings.ToUpper(parts[0])
	params := map[string]string{}
	for _, part := range parts[1:] {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			params[strings.ToUpper(kv[0])] = strings.Trim(kv[1], `"`)
		}
	}
	return icsProperty{name: name, params: params, value: value}, true
}

func unescapeText(v string) string {
	replacer := strings.NewReplacer(`\n`, "\n", `\N`, "\n", `\,`, ",", `\;`, ";", `\\`, `\`)
	return replacer.Replace(v)
}

func parseICSDuration(v string) (time.Duration, error) {
	neg := strings.HasPrefix(v, "-")
	v = strings.TrimPrefix(strings.TrimPrefix(v, "-"), "+")
	if !strings.HasPrefix(v, "P") {
		return 0, fmt.Errorf("invalid duration %q", v)
	}
	v = v[1:]

	var days, hours, mins, secs int
	timePart := false
	numBuf := ""
	for _, r := range v {
		switch {
		case r == 'T':
			timePart = true
		case r >= '0' && r <= '9':
			numBuf += string(r)
		default:
			n, _ := strconv.Atoi(numBuf)
			numBuf = ""
			switch r {
			case 'D':
				days = n
			case 'W':
				days = n * 7
			case 'H':
				hours = n
			case 'M':
				if timePart {
					mins = n
				}
			case 'S':
				secs = n
			}
		}
	}
	d := time.Duration(days)*24*time.Hour + time.Duration(hours)*time.Hour + time.Duration(mins)*time.Minute + time.Duration(secs)*time.Second
	if neg {
		d = -d
	}
	return d, nil
}

// parseICSTime parses a DATE or DATE-TIME ICS value. Returns (time, isAllDay, error).
func parseICSTime(v string, params map[string]string) (time.Time, bool, error) {
	v = strings.TrimSpace(v)
	switch {
	case len(v) == 8:
		t, err := time.ParseInLocation("20060102", v, time.UTC)
		return t, true, err
	case strings.HasSuffix(v, "Z"):
		t, err := time.Parse("20060102T150405Z", v)
		return t, false, err
	default:
		loc := time.UTC
		if tzid, ok := params["TZID"]; ok {
			if l, err := time.LoadLocation(tzid); err == nil {
				loc = l
			}
		}
		t, err := time.ParseInLocation("20060102T150405", v, loc)
		return t, false, err
	}
}
