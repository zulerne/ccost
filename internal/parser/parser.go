package parser

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zulerne/ccost/internal/pricing"
)

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

func (u Usage) IsZero() bool {
	return u.InputTokens == 0 && u.OutputTokens == 0 &&
		u.CacheCreationInputTokens == 0 && u.CacheReadInputTokens == 0
}

type Message struct {
	ID    string `json:"id"`
	Model string `json:"model"`
	Usage Usage  `json:"usage"`
}

type Entry struct {
	Type      string  `json:"type"`
	Timestamp string  `json:"timestamp"`
	CWD       string  `json:"cwd"`
	Message   Message `json:"message"`
}

// Record is a deduplicated assistant entry with parsed time.
type Record struct {
	Time       time.Time
	Model      string
	Project    string
	Input      int
	Output     int
	CacheWrite int
	CacheRead  int
}

// Session represents time spent in a main session file on a single day.
// A session spanning multiple days produces one Session per day.
type Session struct {
	Date     string // YYYY-MM-DD
	Project  string
	Duration time.Duration
}

type Options struct {
	Since   time.Time
	Until   time.Time
	Project string // substring match
}

func claudeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// Parse reads all JSONL files under ~/.claude/projects and returns deduplicated records and sessions.
func Parse(opts Options) ([]Record, []Session, []string, error) {
	dir, err := claudeDir()
	if err != nil {
		return nil, nil, nil, err
	}
	return parseDir(dir, opts)
}

func parseDir(dir string, opts Options) ([]Record, []Session, []string, error) {
	// Main session files: <project>/<uuid>.jsonl
	mainPattern := filepath.Join(dir, "*", "*.jsonl")
	// Subagent files: <project>/<uuid>/subagents/agent-*.jsonl
	subPattern := filepath.Join(dir, "*", "*", "subagents", "*.jsonl")

	mainFiles, err := filepath.Glob(mainPattern)
	if err != nil {
		return nil, nil, nil, err
	}
	subFiles, _ := filepath.Glob(subPattern)

	var allRecords []Record
	var allSessions []Session
	unknownModels := map[string]bool{}

	for _, f := range mainFiles {
		records, sessions, unknown, err := parseFile(f, opts, true)
		if err != nil {
			continue
		}
		allRecords = append(allRecords, records...)
		allSessions = append(allSessions, sessions...)
		for _, m := range unknown {
			unknownModels[m] = true
		}
	}

	for _, f := range subFiles {
		records, _, unknown, err := parseFile(f, opts, false)
		if err != nil {
			continue
		}
		allRecords = append(allRecords, records...)
		for _, m := range unknown {
			unknownModels[m] = true
		}
	}

	sort.Slice(allRecords, func(i, j int) bool {
		return allRecords[i].Time.Before(allRecords[j].Time)
	})

	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].Date < allSessions[j].Date
	})

	var warnings []string
	for m := range unknownModels {
		warnings = append(warnings, "unknown model: "+m)
	}
	sort.Strings(warnings)

	return allRecords, allSessions, warnings, nil
}

func parseTime(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, false
		}
	}
	return t, true
}

// dayBounds tracks min/max timestamps for a single day.
type dayBounds struct {
	min, max time.Time
}

func parseFile(path string, opts Options, isMain bool) ([]Record, []Session, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()

	// First pass: collect entries, deduplicate by message.id (keep max output_tokens).
	// Also track min/max timestamps per day for session duration (main files only).
	best := map[string]Entry{}
	var project string
	days := map[string]*dayBounds{} // date string → bounds

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}

		// Extract project from any entry with CWD.
		if project == "" && e.CWD != "" {
			project = filepath.Base(e.CWD)
		}

		// Track timestamps per day for session duration.
		if isMain && e.Timestamp != "" {
			if t, ok := parseTime(e.Timestamp); ok {
				day := t.Format("2006-01-02")
				b, exists := days[day]
				if !exists {
					b = &dayBounds{min: t, max: t}
					days[day] = b
				} else {
					if t.Before(b.min) {
						b.min = t
					}
					if t.After(b.max) {
						b.max = t
					}
				}
			}
		}

		if e.Type != "assistant" || e.Message.ID == "" {
			continue
		}
		if prev, ok := best[e.Message.ID]; ok {
			if e.Message.Usage.OutputTokens > prev.Message.Usage.OutputTokens {
				best[e.Message.ID] = e
			}
		} else {
			best[e.Message.ID] = e
		}
	}

	if opts.Project != "" && !strings.Contains(strings.ToLower(project), strings.ToLower(opts.Project)) {
		return nil, nil, nil, nil
	}

	var records []Record
	unknownModels := map[string]bool{}

	for _, e := range best {
		// Skip entries with all-zero usage (e.g. <synthetic>).
		if e.Message.Usage.IsZero() {
			continue
		}

		t, ok := parseTime(e.Timestamp)
		if !ok {
			continue
		}

		if !opts.Since.IsZero() && t.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && t.After(opts.Until) {
			continue
		}

		normalized := pricing.NormalizeModel(e.Message.Model)
		if normalized != "" {
			if _, known := pricing.Lookup(normalized); !known {
				unknownModels[normalized] = true
			}
		}

		records = append(records, Record{
			Time:       t,
			Model:      normalized,
			Project:    project,
			Input:      e.Message.Usage.InputTokens,
			Output:     e.Message.Usage.OutputTokens,
			CacheWrite: e.Message.Usage.CacheCreationInputTokens,
			CacheRead:  e.Message.Usage.CacheReadInputTokens,
		})
	}

	// Build per-day sessions for main files.
	var sessions []Session
	if isMain && project != "" {
		for date, b := range days {
			day, _ := time.Parse("2006-01-02", date)
			if !opts.Since.IsZero() && day.Before(opts.Since) {
				continue
			}
			if !opts.Until.IsZero() && day.After(opts.Until) {
				continue
			}
			sessions = append(sessions, Session{
				Date:     date,
				Project:  project,
				Duration: b.max.Sub(b.min),
			})
		}
	}

	var warnings []string
	for m := range unknownModels {
		warnings = append(warnings, m)
	}

	return records, sessions, warnings, nil
}
