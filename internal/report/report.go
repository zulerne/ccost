package report

import (
	"sort"

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
	Cost       float64 // -1 if contains unknown model with non-zero tokens
}

// Report holds aggregated rows and a total.
type Report struct {
	Rows  []Row
	Total Row
}

// ByDate groups records by date, merging all models.
func ByDate(records []parser.Record) Report {
	return aggregate(records, func(r parser.Record) string {
		return r.Time.Format("2006-01-02")
	}, false)
}

// ByDateDetailed groups records by date + model.
func ByDateDetailed(records []parser.Record) Report {
	return aggregate(records, func(r parser.Record) string {
		return r.Time.Format("2006-01-02")
	}, true)
}

// ByProject groups records by project, merging all models.
func ByProject(records []parser.Record) Report {
	return aggregate(records, func(r parser.Record) string {
		return r.Project
	}, false)
}

// ByProjectDetailed groups records by project + model.
func ByProjectDetailed(records []parser.Record) Report {
	return aggregate(records, func(r parser.Record) string {
		return r.Project
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

func aggregate(records []parser.Record, keyFn func(parser.Record) string, detailed bool) Report {
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
	for _, k := range keys {
		a := groups[k]
		if a.hasUnknown {
			a.Cost = -1
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

	return Report{Rows: rows, Total: total}
}
