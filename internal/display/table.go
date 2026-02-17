package display

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/zulerne/ccost/internal/report"
)

func formatNum(n int) string {
	if n == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", n)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func formatCompact(n int) string {
	switch {
	case n >= 1_000_000:
		s := fmt.Sprintf("%.1f", float64(n)/1_000_000)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		return s + "M"
	case n >= 1_000:
		s := fmt.Sprintf("%.1f", float64(n)/1_000)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		return s + "K"
	default:
		return fmt.Sprintf("%d", n)
	}
}

func formatCost(cost float64) string {
	if cost < 0 {
		return "N/A"
	}
	return fmt.Sprintf("$%.2f", cost)
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return ""
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", h, m)
}

func hasModels(rpt report.Report) bool {
	for _, row := range rpt.Rows {
		if row.Model != "" {
			return true
		}
	}
	return false
}

// trimDate removes the "YYYY-" prefix from date-shaped keys.
// Non-date keys are returned as-is.
func trimDate(key string, weekday bool) string {
	if len(key) > 4 && key[4] == '-' {
		s := key[5:]
		if weekday {
			if t, err := time.Parse("2006-01-02", key); err == nil {
				s = t.Format("Mon") + " " + s
			}
		}
		return s
	}
	return key
}

// yearOf extracts the "YYYY" prefix from a date-shaped key, or "".
func yearOf(key string) string {
	if len(key) > 4 && key[4] == '-' {
		return key[:4]
	}
	return ""
}

// Table writes a formatted table to w.
// When exact is true, token counts are shown as full numbers (1,234,567);
// otherwise they use compact notation (1.2M, 34.5K).
func Table(w io.Writer, rpt report.Report, keyHeader string, exact bool, title string) {
	fmtTok := formatCompact
	if exact {
		fmtTok = formatNum
	}

	tw := table.NewWriter()
	tw.SetOutputMirror(w)

	weekly := strings.HasPrefix(title, "Weekly")
	if title != "" {
		tw.SetTitle(text.FgCyan.Sprint(title))
	}

	showModel := hasModels(rpt)

	displayHeader := keyHeader

	if showModel {
		tw.AppendHeader(table.Row{displayHeader, "Model", "Input", "Output", "Write", "Read", "Time", "Cost"})
	} else {
		tw.AppendHeader(table.Row{displayHeader, "Input", "Output", "Write", "Read", "Time", "Cost"})
	}

	years := make(map[string]bool)
	for _, row := range rpt.Rows {
		if y := yearOf(row.Key); y != "" {
			years[y] = true
		}
	}
	multiYear := len(years) > 1

	if showModel {
		prevKey := ""
		prevYear := ""
		for i, row := range rpt.Rows {
			y := yearOf(row.Key)
			if multiYear && y != "" && y != prevYear {
				if i > 0 {
					tw.AppendSeparator()
				}
				tw.AppendRow(table.Row{y}, table.RowConfig{AutoMerge: true})
			}
			if y != "" {
				prevYear = y
			}
			displayKey := trimDate(row.Key, weekly)
			if row.Key == prevKey {
				displayKey = ""
			} else if i > 0 && yearOf(row.Key) == prevYear {
				tw.AppendSeparator()
			}
			prevKey = row.Key

			tw.AppendRow(table.Row{
				displayKey,
				strings.TrimPrefix(row.Model, "claude-"),
				fmtTok(row.Input),
				fmtTok(row.Output),
				fmtTok(row.CacheWrite),
				fmtTok(row.CacheRead),
				formatDuration(row.Duration),
				formatCost(row.Cost),
			})
		}
	} else {
		prevYear := ""
		for _, row := range rpt.Rows {
			if y := yearOf(row.Key); y != "" {
				if multiYear && y != prevYear {
					tw.AppendRow(table.Row{y}, table.RowConfig{AutoMerge: true})
				}
				prevYear = y
			}
			tw.AppendRow(table.Row{
				trimDate(row.Key, weekly),
				fmtTok(row.Input),
				fmtTok(row.Output),
				fmtTok(row.CacheWrite),
				fmtTok(row.CacheRead),
				formatDuration(row.Duration),
				formatCost(row.Cost),
			})
		}
	}

	if showModel {
		tw.AppendFooter(table.Row{
			"TOTAL", "",
			fmtTok(rpt.Total.Input),
			fmtTok(rpt.Total.Output),
			fmtTok(rpt.Total.CacheWrite),
			fmtTok(rpt.Total.CacheRead),
			formatDuration(rpt.Total.Duration),
			formatCost(rpt.Total.Cost),
		})
	} else {
		tw.AppendFooter(table.Row{
			"TOTAL",
			fmtTok(rpt.Total.Input),
			fmtTok(rpt.Total.Output),
			fmtTok(rpt.Total.CacheWrite),
			fmtTok(rpt.Total.CacheRead),
			formatDuration(rpt.Total.Duration),
			formatCost(rpt.Total.Cost),
		})
	}

	// Right-align numeric columns. Column numbers shift when Model is present.
	numStart := 2
	if showModel {
		numStart = 3
	}
	var colConfigs []table.ColumnConfig
	for i := numStart; i <= numStart+5; i++ {
		colConfigs = append(colConfigs, table.ColumnConfig{
			Number:      i,
			Align:       text.AlignRight,
			AlignHeader: text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	}
	tw.SetColumnConfigs(colConfigs)

	tw.SetStyle(table.StyleRounded)
	tw.Style().Color.Header = text.Colors{text.FgCyan}
	tw.Style().Color.Footer = text.Colors{text.FgYellow}
	tw.Style().Options.DoNotColorBordersAndSeparators = true

	isYearRow := func(row table.Row) bool {
		if len(row) == 0 {
			return false
		}
		s, ok := row[0].(string)
		return ok && len(s) == 4 && s[0] >= '1' && s[0] <= '9'
	}

	if showModel {
		tw.SetRowPainter(func(row table.Row) text.Colors {
			if isYearRow(row) {
				return text.Colors{text.FgCyan}
			}
			return nil
		})
	} else {
		rowIdx := 0
		tw.SetRowPainter(func(row table.Row) text.Colors {
			if isYearRow(row) {
				return text.Colors{text.FgCyan}
			}
			rowIdx++
			if rowIdx%2 == 0 {
				return text.Colors{text.Faint}
			}
			return nil
		})
	}

	tw.Render()
}
