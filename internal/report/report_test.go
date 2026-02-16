package report

import (
	"math"
	"testing"
	"time"

	"github.com/zulerne/ccost/internal/parser"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}

func TestByDateMergesModels(t *testing.T) {
	records := []parser.Record{
		{
			Time:    time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
			Model:   "claude-opus-4-6",
			Project: "proj",
			Input:   1000,
			Output:  500,
		},
		{
			Time:    time.Date(2026, 2, 14, 11, 0, 0, 0, time.UTC),
			Model:   "claude-sonnet-4-5",
			Project: "proj",
			Input:   2000,
			Output:  1000,
		},
		{
			Time:    time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
			Model:   "claude-sonnet-4-5",
			Project: "proj",
			Input:   500,
			Output:  200,
		},
	}

	rpt := ByDate(records, nil)

	// Two models on 2026-02-14 should merge into one row.
	if len(rpt.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rpt.Rows))
	}

	if rpt.Rows[0].Key != "2026-02-14" {
		t.Errorf("expected key '2026-02-14', got %q", rpt.Rows[0].Key)
	}
	if rpt.Rows[0].Input != 3000 {
		t.Errorf("expected input 3000, got %d", rpt.Rows[0].Input)
	}
	if rpt.Rows[0].Output != 1500 {
		t.Errorf("expected output 1500, got %d", rpt.Rows[0].Output)
	}

	if rpt.Total.Input != 3500 {
		t.Errorf("expected total input 3500, got %d", rpt.Total.Input)
	}
	if rpt.Total.Cost < 0 {
		t.Error("expected non-negative total cost")
	}
}

func TestByProject(t *testing.T) {
	records := []parser.Record{
		{
			Time:    time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
			Model:   "claude-opus-4-6",
			Project: "alpha",
			Input:   100,
			Output:  50,
		},
		{
			Time:    time.Date(2026, 2, 14, 11, 0, 0, 0, time.UTC),
			Model:   "claude-opus-4-6",
			Project: "beta",
			Input:   200,
			Output:  100,
		},
	}

	rpt := ByProject(records, nil)
	if len(rpt.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rpt.Rows))
	}
	if rpt.Rows[0].Key != "alpha" {
		t.Errorf("expected 'alpha', got %q", rpt.Rows[0].Key)
	}
}

func TestByDateDetailedSplitsModels(t *testing.T) {
	records := []parser.Record{
		{
			Time:    time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
			Model:   "claude-opus-4-6",
			Project: "proj",
			Input:   1000,
			Output:  500,
		},
		{
			Time:    time.Date(2026, 2, 14, 11, 0, 0, 0, time.UTC),
			Model:   "claude-sonnet-4-5",
			Project: "proj",
			Input:   2000,
			Output:  1000,
		},
	}

	rpt := ByDateDetailed(records, nil)

	// Same date but different models should produce 2 rows.
	if len(rpt.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rpt.Rows))
	}
	if rpt.Rows[0].Model != "claude-opus-4-6" {
		t.Errorf("expected first model 'claude-opus-4-6', got %q", rpt.Rows[0].Model)
	}
	if rpt.Rows[1].Model != "claude-sonnet-4-5" {
		t.Errorf("expected second model 'claude-sonnet-4-5', got %q", rpt.Rows[1].Model)
	}
	if rpt.Total.Input != 3000 {
		t.Errorf("expected total input 3000, got %d", rpt.Total.Input)
	}
}

func TestUnknownModelCost(t *testing.T) {
	records := []parser.Record{
		{
			Time:   time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
			Model:  "unknown-model",
			Input:  100,
			Output: 50,
		},
	}
	rpt := ByDate(records, nil)
	if rpt.Rows[0].Cost != -1 {
		t.Errorf("expected cost -1 for unknown model, got %f", rpt.Rows[0].Cost)
	}
}

func TestTotalCostKnown(t *testing.T) {
	records := []parser.Record{
		{
			Time:   time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
			Model:  "claude-opus-4-6",
			Input:  1000,
			Output: 500,
		},
	}
	rpt := ByDate(records, nil)
	// 1000*5/1M + 500*25/1M = 0.005 + 0.0125 = 0.0175
	if !almostEqual(rpt.Total.Cost, 0.0175) {
		t.Errorf("expected total cost ~0.0175, got %f", rpt.Total.Cost)
	}
}

func TestByDateWithDuration(t *testing.T) {
	records := []parser.Record{
		{
			Time:   time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
			Model:  "claude-opus-4-6",
			Input:  1000,
			Output: 500,
		},
	}
	sessions := []parser.Session{
		{Date: "2026-02-14", Project: "proj", Duration: 30 * time.Minute},
		{Date: "2026-02-14", Project: "proj", Duration: 15 * time.Minute},
	}

	rpt := ByDate(records, sessions)
	if rpt.Rows[0].Duration != 45*time.Minute {
		t.Errorf("expected duration 45m, got %v", rpt.Rows[0].Duration)
	}
	if rpt.Total.Duration != 45*time.Minute {
		t.Errorf("expected total duration 45m, got %v", rpt.Total.Duration)
	}
}

func TestByDateDetailedDuration(t *testing.T) {
	records := []parser.Record{
		{
			Time:    time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
			Model:   "claude-opus-4-6",
			Project: "proj",
			Input:   1000,
			Output:  500,
		},
		{
			Time:    time.Date(2026, 2, 14, 11, 0, 0, 0, time.UTC),
			Model:   "claude-sonnet-4-5",
			Project: "proj",
			Input:   2000,
			Output:  1000,
		},
	}
	sessions := []parser.Session{
		{Date: "2026-02-14", Project: "proj", Duration: 1 * time.Hour},
	}

	rpt := ByDateDetailed(records, sessions)
	// Only first row of the key group should have duration.
	if rpt.Rows[0].Duration != 1*time.Hour {
		t.Errorf("expected first row duration 1h, got %v", rpt.Rows[0].Duration)
	}
	if rpt.Rows[1].Duration != 0 {
		t.Errorf("expected second row duration 0, got %v", rpt.Rows[1].Duration)
	}
	if rpt.Total.Duration != 1*time.Hour {
		t.Errorf("expected total duration 1h, got %v", rpt.Total.Duration)
	}
}

func TestByProjectWithDuration(t *testing.T) {
	records := []parser.Record{
		{
			Time:    time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
			Model:   "claude-opus-4-6",
			Project: "alpha",
			Input:   100,
			Output:  50,
		},
		{
			Time:    time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
			Model:   "claude-opus-4-6",
			Project: "beta",
			Input:   200,
			Output:  100,
		},
	}
	sessions := []parser.Session{
		{Date: "2026-02-14", Project: "alpha", Duration: 20 * time.Minute},
		{Date: "2026-02-15", Project: "beta", Duration: 40 * time.Minute},
	}

	rpt := ByProject(records, sessions)
	if rpt.Rows[0].Duration != 20*time.Minute {
		t.Errorf("expected alpha duration 20m, got %v", rpt.Rows[0].Duration)
	}
	if rpt.Rows[1].Duration != 40*time.Minute {
		t.Errorf("expected beta duration 40m, got %v", rpt.Rows[1].Duration)
	}
	if rpt.Total.Duration != 60*time.Minute {
		t.Errorf("expected total duration 60m, got %v", rpt.Total.Duration)
	}
}
