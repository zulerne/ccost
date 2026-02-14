package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/zulerne/ccost/internal/display"
	"github.com/zulerne/ccost/internal/parser"
	"github.com/zulerne/ccost/internal/report"
)

func main() {
	var (
		sinceStr  string
		untilStr  string
		project   string
		byProject bool
		models    bool
		jsonOut   bool
	)

	flag.StringVar(&sinceStr, "since", "", "start date (YYYY-MM-DD)")
	flag.StringVar(&untilStr, "until", "", "end date (YYYY-MM-DD), inclusive")
	flag.StringVar(&project, "project", "", "filter by project name (substring)")
	flag.BoolVar(&byProject, "by-project", false, "group by project instead of date")
	flag.BoolVar(&models, "models", false, "show per-model breakdown")
	flag.BoolVar(&jsonOut, "json", false, "output as JSON")
	flag.Parse()

	opts := parser.Options{
		Project: project,
	}

	if sinceStr != "" {
		t, err := time.Parse("2006-01-02", sinceStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --since date: %v\n", err)
			os.Exit(1)
		}
		opts.Since = t
	}

	if untilStr != "" {
		t, err := time.Parse("2006-01-02", untilStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --until date: %v\n", err)
			os.Exit(1)
		}
		// Make until inclusive: set to end of that day.
		opts.Until = t.Add(24*time.Hour - time.Nanosecond)
	}

	records, warnings, err := parser.Parse(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	if len(records) == 0 {
		fmt.Fprintln(os.Stderr, "no records found")
		os.Exit(0)
	}

	var rpt report.Report
	keyHeader := "Date"
	if byProject {
		if models {
			rpt = report.ByProjectDetailed(records)
		} else {
			rpt = report.ByProject(records)
		}
		keyHeader = "Project"
	} else {
		if models {
			rpt = report.ByDateDetailed(records)
		} else {
			rpt = report.ByDate(records)
		}
	}

	if jsonOut {
		if err := display.JSON(os.Stdout, rpt); err != nil {
			fmt.Fprintf(os.Stderr, "error writing JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		display.Table(os.Stdout, rpt, keyHeader)
	}
}
