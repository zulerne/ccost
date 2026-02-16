package report

import (
	"sort"
	"time"

	"github.com/zulerne/ccost/internal/parser"
	"github.com/zulerne/ccost/internal/pricing"
)

// Row is a single aggregated line in the report.
type Row struct {
	Key        string // date (YYYY-MM-DD) or project name
	Model      string // populated only in detailed (--models) mode
	Input      int
	Output     int
	CacheWrite int
	CacheRead  int
	Cost       float64       // -1 if contains unknown model with non-zero tokens
	Duration   time.Duration // session time; zero for per-model detail rows
}

// Report holds aggregated rows and a total.
type Report struct {
	Rows  []Row
	Total Row
}

// ByDate groups records by date, merging all models.
func ByDate(records []parser.Record, sessions []parser.Session) Report {
	return aggregate(records, sessions, func(r parser.Record) string {
		return r.Time.Format("2006-01-02")
	}, func(s parser.Session) string {
		return s.Date
	}, false)
}

// ByDateDetailed groups records by date + model.
func ByDateDetailed(records []parser.Record, sessions []parser.Session) Report {
	return aggregate(records, sessions, func(r parser.Record) string {
		return r.Time.Format("2006-01-02")
	}, func(s parser.Session) string {
		return s.Date
	}, true)
}

// ByProject groups records by project, merging all models.
func ByProject(records []parser.Record, sessions []parser.Session) Report {
	return aggregate(records, sessions, func(r parser.Record) string {
		return r.Project
	}, func(s parser.Session) string {
		return s.Project
	}, false)
}

// ByProjectDetailed groups records by project + model.
func ByProjectDetailed(records []parser.Record, sessions []parser.Session) Report {
	return aggregate(records, sessions, func(r parser.Record) string {
		return r.Project
	}, func(s parser.Session) string {
		return s.Project
	}, true)
}

type groupKey struct {
	key   string
	model string
}

type accum struct {
	Row
	hasUnknown bool
}

func aggregate(
	records []parser.Record,
	sessions []parser.Session,
	keyFn func(parser.Record) string,
	sessionKeyFn func(parser.Session) string,
	detailed bool,
) Report {
	groups := map[groupKey]*accum{}
	var keys []groupKey

	for _, r := range records {
		k := groupKey{key: keyFn(r)}
		if detailed {
			k.model = r.Model
		}
		a, ok := groups[k]
		if !ok {
			a = &accum{Row: Row{Key: k.key, Model: k.model}}
			groups[k] = a
			keys = append(keys, k)
		}
		a.Input += r.Input
		a.Output += r.Output
		a.CacheWrite += r.CacheWrite
		a.CacheRead += r.CacheRead

		c := pricing.Cost(r.Model, r.Input, r.Output, r.CacheWrite, r.CacheRead)
		if c >= 0 {
			a.Cost += c
		} else {
			a.hasUnknown = true
		}
	}

	// Aggregate session durations per key (not per model).
	durations := map[string]time.Duration{}
	for _, s := range sessions {
		durations[sessionKeyFn(s)] += s.Duration
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].key != keys[j].key {
			return keys[i].key < keys[j].key
		}
		return keys[i].model < keys[j].model
	})

	var total Row
	total.Key = "TOTAL"
	totalHasUnknown := false

	rows := make([]Row, 0, len(keys))
	seen := map[string]bool{}
	for _, k := range keys {
		a := groups[k]
		if a.hasUnknown {
			a.Cost = -1
		}
		// Assign duration: once per key group (first row only in detailed mode).
		if !detailed || !seen[k.key] {
			a.Duration = durations[k.key]
			seen[k.key] = true
		}
		rows = append(rows, a.Row)

		total.Input += a.Input
		total.Output += a.Output
		total.CacheWrite += a.CacheWrite
		total.CacheRead += a.CacheRead
		if a.hasUnknown {
			totalHasUnknown = true
		} else {
			total.Cost += a.Cost
		}
	}

	if totalHasUnknown {
		total.Cost = -1
	}

	for _, d := range durations {
		total.Duration += d
	}

	return Report{Rows: rows, Total: total}
}
