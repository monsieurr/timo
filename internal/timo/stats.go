package timo

import "time"

type Day struct {
	Date         string `json:"date"`
	FocusMinutes int    `json:"focus_minutes"`
	Sessions     int    `json:"sessions"`
}
type Stats struct {
	FocusSessions int   `json:"focus_sessions"`
	FocusMinutes  int   `json:"focus_minutes"`
	BreakSessions int   `json:"break_sessions"`
	TodayMinutes  int   `json:"today_minutes"`
	Streak        int   `json:"streak"`
	Days          []Day `json:"days"`
}

// ComputeStats groups sessions by completion date, in the chosen local time zone.
func ComputeStats(s State, now time.Time, loc *time.Location) Stats {
	out := Stats{Days: []Day{}}
	count := map[string]*Day{}
	today := now.In(loc).Format("2006-01-02")
	for _, entry := range s.History {
		date := entry.CompletedAt.In(loc).Format("2006-01-02")
		if entry.Phase != PhaseFocus {
			out.BreakSessions++
			continue
		}
		out.FocusSessions++
		mins := int(entry.DurationSeconds / 60)
		out.FocusMinutes += mins
		if date == today {
			out.TodayMinutes += mins
		}
		if count[date] == nil {
			count[date] = &Day{Date: date}
		}
		count[date].FocusMinutes += mins
		count[date].Sessions++
	}
	for d := 0; d < 7; d++ {
		dt := now.In(loc).AddDate(0, 0, d-6).Format("2006-01-02")
		if day := count[dt]; day != nil {
			out.Days = append(out.Days, *day)
		} else {
			out.Days = append(out.Days, Day{Date: dt})
		}
	}
	for d := 0; ; d++ {
		dt := now.In(loc).AddDate(0, 0, -d).Format("2006-01-02")
		if d == 0 && count[dt] == nil {
			continue
		}
		if count[dt] == nil {
			break
		}
		out.Streak++
	}
	return out
}
