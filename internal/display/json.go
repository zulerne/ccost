package display

import (
	"encoding/json"
	"io"
	"math"

	"github.com/zulerne/ccost/internal/report"
)

type jsonRow struct {
	Key        string  `json:"key"`
	Model      string  `json:"model,omitempty"`
	Input      int     `json:"input_tokens"`
	Output     int     `json:"output_tokens"`
	CacheWrite int     `json:"cache_write_tokens"`
	CacheRead  int     `json:"cache_read_tokens"`
	Cost       float64 `json:"cost"`
}

type jsonReport struct {
	Rows  []jsonRow `json:"rows"`
	Total jsonRow   `json:"total"`
}

func roundCost(c float64) float64 {
	if c < 0 {
		return c
	}
	return math.Round(c*100) / 100
}

// JSON writes the report as JSON to w.
func JSON(w io.Writer, rpt report.Report) error {
	jr := jsonReport{
		Rows: make([]jsonRow, len(rpt.Rows)),
		Total: jsonRow{
			Key:        rpt.Total.Key,
			Input:      rpt.Total.Input,
			Output:     rpt.Total.Output,
			CacheWrite: rpt.Total.CacheWrite,
			CacheRead:  rpt.Total.CacheRead,
			Cost:       roundCost(rpt.Total.Cost),
		},
	}

	for i, r := range rpt.Rows {
		jr.Rows[i] = jsonRow{
			Key:        r.Key,
			Model:      r.Model,
			Input:      r.Input,
			Output:     r.Output,
			CacheWrite: r.CacheWrite,
			CacheRead:  r.CacheRead,
			Cost:       roundCost(r.Cost),
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jr)
}
