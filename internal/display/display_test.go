package display

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/zulerne/ccost/internal/report"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func sampleReport() report.Report {
	return report.Report{
		Rows: []report.Row{
			{
				Key:        "2026-02-14",
				Input:      19290,
				Output:     5561,
				CacheWrite: 130693,
				CacheRead:  6430159,
				Cost:       12.80,
				Duration:   2*time.Hour + 15*time.Minute,
			},
		},
		Total: report.Row{
			Key:        "TOTAL",
			Input:      19290,
			Output:     5561,
			CacheWrite: 130693,
			CacheRead:  6430159,
			Cost:       12.80,
			Duration:   2*time.Hour + 15*time.Minute,
		},
	}
}

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, sampleReport(), "Date", false)
	out := strings.ToUpper(stripANSI(buf.String()))

	if !strings.Contains(out, "DATE") {
		t.Errorf("expected 'DATE' header in output:\n%s", out)
	}
	if !strings.Contains(out, "TIME") {
		t.Errorf("expected 'TIME' header in output:\n%s", out)
	}
	if !strings.Contains(out, "19.3K") {
		t.Errorf("expected compact number '19.3K' in output:\n%s", out)
	}
	if !strings.Contains(out, "$12.80") {
		t.Errorf("expected cost '$12.80' in output:\n%s", out)
	}
	if !strings.Contains(out, "2H15M") {
		t.Errorf("expected duration '2H15M' in output:\n%s", out)
	}
	if !strings.Contains(out, "TOTAL") {
		t.Error("expected TOTAL row")
	}
}

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, sampleReport()); err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	rows, ok := result["rows"].([]interface{})
	if !ok || len(rows) != 1 {
		t.Fatalf("expected 1 row, got %v", result["rows"])
	}

	row := rows[0].(map[string]interface{})
	if row["duration_seconds"] != float64(8100) {
		t.Errorf("expected duration_seconds 8100, got %v", row["duration_seconds"])
	}

	total, ok := result["total"].(map[string]interface{})
	if !ok {
		t.Fatal("expected total object")
	}
	if total["key"] != "TOTAL" {
		t.Errorf("expected total key 'TOTAL', got %v", total["key"])
	}
	if total["duration_seconds"] != float64(8100) {
		t.Errorf("expected total duration_seconds 8100, got %v", total["duration_seconds"])
	}
}

func TestFormatNum(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{123, "123"},
		{1234, "1,234"},
		{1234567, "1,234,567"},
	}
	for _, tt := range tests {
		got := formatNum(tt.in)
		if got != tt.want {
			t.Errorf("formatNum(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatCostRounding(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{12.8022, "$12.80"},
		{0.4523, "$0.45"},
		{0.0, "$0.00"},
		{-1, "N/A"},
	}
	for _, tt := range tests {
		got := formatCost(tt.in)
		if got != tt.want {
			t.Errorf("formatCost(%f) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, ""},
		{5 * time.Minute, "0h05m"},
		{90 * time.Minute, "1h30m"},
		{2*time.Hour + 15*time.Minute, "2h15m"},
		{10*time.Hour + 3*time.Minute, "10h03m"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.in)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatCompact(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1K"},
		{1234, "1.2K"},
		{19290, "19.3K"},
		{130693, "130.7K"},
		{1000000, "1M"},
		{6430159, "6.4M"},
		{1234567, "1.2M"},
		{10000000, "10M"},
	}
	for _, tt := range tests {
		got := formatCompact(tt.in)
		if got != tt.want {
			t.Errorf("formatCompact(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTableExact(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, sampleReport(), "Date", true)
	out := strings.ToUpper(stripANSI(buf.String()))

	if !strings.Contains(out, "19,290") {
		t.Errorf("expected exact number '19,290' in output:\n%s", out)
	}
}

func TestTableWithModels(t *testing.T) {
	rpt := report.Report{
		Rows: []report.Row{
			{Key: "2026-02-14", Model: "claude-opus-4-6", Input: 1000, Output: 500, Cost: 0.05},
			{Key: "2026-02-14", Model: "claude-sonnet-4-5", Input: 2000, Output: 1000, Cost: 0.02},
		},
		Total: report.Row{Key: "TOTAL", Input: 3000, Output: 1500, Cost: 0.07},
	}
	var buf bytes.Buffer
	Table(&buf, rpt, "Date", false)
	out := stripANSI(buf.String())

	if !strings.Contains(strings.ToUpper(out), "MODEL") {
		t.Error("expected 'MODEL' column header when models are present")
	}
	if !strings.Contains(out, "claude-opus-4-6") {
		t.Error("expected 'claude-opus-4-6' in output")
	}
	if !strings.Contains(out, "claude-sonnet-4-5") {
		t.Error("expected 'claude-sonnet-4-5' in output")
	}
}

func TestTableByProject(t *testing.T) {
	rpt := report.Report{
		Rows: []report.Row{
			{Key: "myproject", Input: 100, Output: 50, Cost: 0.005},
		},
		Total: report.Row{Key: "TOTAL", Input: 100, Output: 50, Cost: 0.005},
	}
	var buf bytes.Buffer
	Table(&buf, rpt, "Project", false)
	if !strings.Contains(strings.ToUpper(stripANSI(buf.String())), "PROJECT") {
		t.Error("expected 'PROJECT' header")
	}
}
