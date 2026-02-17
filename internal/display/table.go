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

// commonYear returns the year prefix (e.g. "2026") if all keys share it,
// or "" if keys span multiple years or aren't date-shaped.
func commonYear(rows []report.Row) string {
	if len(rows) == 0 {
		return ""
	}
	year := ""
	for _, row := range rows {
		if len(row.Key) < 5 || row.Key[4] != '-' {
			return ""
		}
		y := row.Key[:4]
		if year == "" {
			year = y
		} else if y != year {
			return ""
		}
	}
	return year
}

// Table writes a formatted table to w.
// When exact is true, token counts are shown as full numbers (1,234,567);
// otherwise they use compact notation (1.2M, 34.5K).
func Table(w io.Writer, rpt report.Report, keyHeader string, exact bool) {
	fmtTok := formatCompact
	if exact {
		fmtTok = formatNum
	}

	tw := table.NewWriter()
	tw.SetOutputMirror(w)

	showModel := hasModels(rpt)

	// If all dates share the same year, show it in the header and trim from rows.
	year := commonYear(rpt.Rows)
	displayHeader := keyHeader
	if year != "" {
		displayHeader = keyHeader + " (" + year + ")"
	}

	if showModel {
		tw.AppendHeader(table.Row{displayHeader, "Model", "Input", "Output", "Cache Write", "Cache Read", "Time", "Cost"})
	} else {
		tw.AppendHeader(table.Row{displayHeader, "Input", "Output", "Cache Write", "Cache Read", "Time", "Cost"})
	}

	if showModel {
		prevKey := ""
		for i, row := range rpt.Rows {
			displayKey := row.Key
			if year != "" {
				displayKey = strings.TrimPrefix(displayKey, year+"-")
			}
			if row.Key == prevKey {
				displayKey = ""
			} else if i > 0 {
				tw.AppendSeparator()
			}
			prevKey = row.Key

			tw.AppendRow(table.Row{
				displayKey,
				row.Model,
				fmtTok(row.Input),
				fmtTok(row.Output),
				fmtTok(row.CacheWrite),
				fmtTok(row.CacheRead),
				formatDuration(row.Duration),
				formatCost(row.Cost),
			})
		}
	} else {
		for _, row := range rpt.Rows {
			displayKey := row.Key
			if year != "" {
				displayKey = strings.TrimPrefix(displayKey, year+"-")
			}
			tw.AppendRow(table.Row{
				displayKey,
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

	tw.Render()
}
