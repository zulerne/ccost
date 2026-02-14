package pricing

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}

func TestCostOpus46(t *testing.T) {
	// 1000 * $5/1M + 500 * $25/1M + 2000 * $10/1M + 10000 * $0.50/1M
	// = 0.005 + 0.0125 + 0.02 + 0.005 = 0.0425
	got := Cost("claude-opus-4-6", 1000, 500, 2000, 10000)
	if !almostEqual(got, 0.0425) {
		t.Errorf("expected 0.0425, got %f", got)
	}
}

func TestCostSonnetWithSuffix(t *testing.T) {
	got := Cost("claude-sonnet-4-5-20250929", 1000, 1000, 0, 0)
	// 1000*3/1M + 1000*15/1M = 0.003 + 0.015 = 0.018
	if !almostEqual(got, 0.018) {
		t.Errorf("expected 0.018, got %f", got)
	}
}

func TestCostHaiku45(t *testing.T) {
	// 1000 * $1/1M + 500 * $5/1M + 2000 * $2/1M + 10000 * $0.10/1M
	// = 0.001 + 0.0025 + 0.004 + 0.001 = 0.0085
	got := Cost("claude-haiku-4-5-20251001", 1000, 500, 2000, 10000)
	if !almostEqual(got, 0.0085) {
		t.Errorf("expected 0.0085, got %f", got)
	}
}

func TestCostUnknownModel(t *testing.T) {
	got := Cost("unknown-model", 1000, 1000, 0, 0)
	if got != -1 {
		t.Errorf("expected -1 for unknown model, got %f", got)
	}
}

func TestLookup(t *testing.T) {
	_, ok := Lookup("claude-opus-4-6")
	if !ok {
		t.Error("expected opus to be found")
	}

	_, ok = Lookup("claude-haiku-4-5-20251001")
	if !ok {
		t.Error("expected haiku with date suffix to be found")
	}

	_, ok = Lookup("nonexistent")
	if ok {
		t.Error("expected nonexistent to not be found")
	}
}

func TestNormalizeModel(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"claude-opus-4-6", "claude-opus-4-6"},
		{"claude-sonnet-4-5-20250929", "claude-sonnet-4-5"},
		{"claude-haiku-4-5-20251001", "claude-haiku-4-5"},
		{"unknown-model", "unknown-model"},
	}
	for _, tt := range tests {
		got := NormalizeModel(tt.in)
		if got != tt.want {
			t.Errorf("NormalizeModel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
