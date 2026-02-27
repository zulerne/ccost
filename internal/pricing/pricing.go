package pricing

import "regexp"

// Price per 1M tokens for each token type.
type ModelPricing struct {
	Input      float64
	Output     float64
	CacheWrite float64 // 1-hour ephemeral cache write
	CacheRead  float64
}

// Source: https://platform.claude.com/docs/en/about-claude/pricing
// Claude Code uses 1-hour ephemeral cache: write = 2x input, read = 0.1x input
var models = map[string]ModelPricing{
	"claude-opus-4-6":   {Input: 5.0, Output: 25.0, CacheWrite: 10.0, CacheRead: 0.50},
	"claude-opus-4-5":   {Input: 5.0, Output: 25.0, CacheWrite: 10.0, CacheRead: 0.50},
	"claude-opus-4-1":   {Input: 15.0, Output: 75.0, CacheWrite: 30.0, CacheRead: 1.50},
	"claude-opus-4":     {Input: 15.0, Output: 75.0, CacheWrite: 30.0, CacheRead: 1.50},
	"claude-sonnet-4-6": {Input: 3.0, Output: 15.0, CacheWrite: 6.0, CacheRead: 0.30},
	"claude-sonnet-4-5": {Input: 3.0, Output: 15.0, CacheWrite: 6.0, CacheRead: 0.30},
	"claude-sonnet-4":   {Input: 3.0, Output: 15.0, CacheWrite: 6.0, CacheRead: 0.30},
	"claude-sonnet-3-7": {Input: 3.0, Output: 15.0, CacheWrite: 6.0, CacheRead: 0.30},
	"claude-haiku-4-5":  {Input: 1.0, Output: 5.0, CacheWrite: 2.0, CacheRead: 0.10},
	"claude-haiku-3-5":  {Input: 0.80, Output: 4.0, CacheWrite: 1.60, CacheRead: 0.08},
	"claude-opus-3":     {Input: 15.0, Output: 75.0, CacheWrite: 30.0, CacheRead: 1.50},
	"claude-haiku-3":    {Input: 0.25, Output: 1.25, CacheWrite: 0.50, CacheRead: 0.03},
}

var dateSuffix = regexp.MustCompile(`-\d{8}$`)

// NormalizeModel strips date suffixes like -20250929 from model names.
func NormalizeModel(model string) string {
	return dateSuffix.ReplaceAllString(model, "")
}

// Lookup returns the pricing for a model and whether it was found.
func Lookup(model string) (ModelPricing, bool) {
	p, ok := models[NormalizeModel(model)]
	return p, ok
}

// Cost calculates the total cost in USD for the given token counts.
// Returns -1 if the model is unknown.
func Cost(model string, input, output, cacheWrite, cacheRead int) float64 {
	p, ok := models[NormalizeModel(model)]
	if !ok {
		return -1
	}
	return (float64(input)*p.Input +
		float64(output)*p.Output +
		float64(cacheWrite)*p.CacheWrite +
		float64(cacheRead)*p.CacheRead) / 1_000_000
}
