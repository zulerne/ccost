package main

import (
	"fmt"
	"os"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/zulerne/ccost/internal/display"
	"github.com/zulerne/ccost/internal/parser"
	"github.com/zulerne/ccost/internal/report"
)

var version = "dev"

func main() {
	var (
		sinceStr   string
		untilStr   string
		project    string
		byProject  bool
		models     bool
		exact      bool
		jsonOut    bool
		versionOut bool
	)

	flag.StringVarP(&sinceStr, "since", "s", "", "start date (YYYY-MM-DD)")
	flag.StringVarP(&untilStr, "until", "u", "", "end date (YYYY-MM-DD), inclusive")
	flag.StringVarP(&project, "project", "p", "", "filter by project name (substring)")
	flag.BoolVarP(&byProject, "by-project", "b", false, "group by project instead of date")
	flag.BoolVarP(&models, "models", "m", false, "show per-model breakdown")
	flag.BoolVarP(&exact, "exact", "e", false, "show exact token counts instead of compact (K/M)")
	flag.BoolVar(&jsonOut, "json", false, "output as JSON")
	flag.BoolVarP(&versionOut, "version", "v", false, "print version and exit")
	flag.Parse()

	if versionOut {
		fmt.Println("ccost " + version)
		os.Exit(0)
	}

	weeklyMode := sinceStr == "" && untilStr == ""

	opts := parser.Options{
		Project: project,
	}

	if weeklyMode {
		now := time.Now()
		sevenDaysAgo := now.AddDate(0, 0, -6)
		opts.Since = time.Date(sevenDaysAgo.Year(), sevenDaysAgo.Month(), sevenDaysAgo.Day(), 0, 0, 0, 0, now.Location())
	}

	if sinceStr != "" {
		t, err := time.ParseInLocation("2006-01-02", sinceStr, time.Local)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --since date: %v\n", err)
			os.Exit(1)
		}
		opts.Since = t
	}

	if untilStr != "" {
		t, err := time.ParseInLocation("2006-01-02", untilStr, time.Local)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --until date: %v\n", err)
			os.Exit(1)
		}
		// Make until inclusive: set to end of that day.
		opts.Until = t.Add(24*time.Hour - time.Nanosecond)
	}

	records, sessions, warnings, err := parser.Parse(opts)
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
		keyHeader = "Project"
	}
	var title string
	if weeklyMode {
		now := time.Now()
		title = fmt.Sprintf("Weekly · %s – %s", opts.Since.Format("Jan 02"), now.Format("Jan 02"))
	} else if sinceStr != "" && untilStr != "" {
		title = fmt.Sprintf("Range · %s – %s", opts.Since.Format("Jan 02"), opts.Until.Format("Jan 02"))
	} else if sinceStr != "" {
		title = fmt.Sprintf("Since · %s", opts.Since.Format("Jan 02"))
	} else if untilStr != "" {
		title = fmt.Sprintf("Until · %s", opts.Until.Format("Jan 02"))
	}
	if byProject {
		if models {
			rpt = report.ByProjectDetailed(records, sessions)
		} else {
			rpt = report.ByProject(records, sessions)
		}
	} else {
		if models {
			rpt = report.ByDateDetailed(records, sessions)
		} else {
			rpt = report.ByDate(records, sessions)
		}
	}

	if jsonOut {
		if err := display.JSON(os.Stdout, rpt); err != nil {
			fmt.Fprintf(os.Stderr, "error writing JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		display.Table(os.Stdout, rpt, keyHeader, exact, title)
	}
}
