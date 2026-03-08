package parser

import (
	"bufio"
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
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
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// Parse reads all JSONL files under ~/.claude/projects and returns deduplicated records and sessions.
func Parse(opts Options) ([]Record, []Session, []string, error) {
	dir, err := claudeDir()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("finding claude directory: %w", err)
	}
	return parseDir(dir, opts)
}

type fileJob struct {
	path   string
	isMain bool
}

type fileResult struct {
	records  []Record
	sessions []Session
	unknown  []string
	cwd      string // full CWD path for project disambiguation
	err      error  // non-nil if parseFile failed
}

func parseDir(dir string, opts Options) ([]Record, []Session, []string, error) {
	// Main session files: <project>/<uuid>.jsonl
	mainPattern := filepath.Join(dir, "*", "*.jsonl")
	// Subagent files: <project>/<uuid>/subagents/agent-*.jsonl
	subPattern := filepath.Join(dir, "*", "*", "subagents", "*.jsonl")

	mainFiles, err := filepath.Glob(mainPattern)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("globbing session files: %w", err)
	}
	// Pattern is hardcoded; filepath.Glob only errors on malformed patterns.
	subFiles, _ := filepath.Glob(subPattern)

	jobs := make([]fileJob, 0, len(mainFiles)+len(subFiles))
	for _, f := range mainFiles {
		jobs = append(jobs, fileJob{path: f, isMain: true})
	}
	for _, f := range subFiles {
		jobs = append(jobs, fileJob{path: f, isMain: false})
	}

	results := make([]fileResult, len(jobs))
	workers := min(runtime.NumCPU(), len(jobs))

	var wg sync.WaitGroup
	ch := make(chan int, len(jobs))
	for i := range jobs {
		ch <- i
	}
	close(ch)

	for range workers {
		wg.Go(func() {
			for i := range ch {
				records, sessions, unknown, cwd, err := parseFile(jobs[i].path, opts, jobs[i].isMain)
				if err != nil {
					results[i] = fileResult{err: err}
					continue
				}
				results[i] = fileResult{records: records, sessions: sessions, unknown: unknown, cwd: cwd}
			}
		})
	}
	wg.Wait()

	// Build baseName → set of unique full CWDs for disambiguation.
	cwdsByBase := map[string]map[string]bool{}
	for _, r := range results {
		if r.cwd == "" {
			continue
		}
		base := filepath.Base(r.cwd)
		if cwdsByBase[base] == nil {
			cwdsByBase[base] = map[string]bool{}
		}
		cwdsByBase[base][r.cwd] = true
	}
	displayNames := disambiguateProjects(cwdsByBase)

	// Merge results: apply disambiguated project names and project filter.
	var allRecords []Record
	projectFilter := strings.ToLower(opts.Project)
	var allSessions []Session
	var fileErrors []string
	unknownModels := map[string]bool{}

	for _, r := range results {
		if r.err != nil {
			fileErrors = append(fileErrors, "skipped file: "+r.err.Error())
			continue
		}
		name := displayNames[r.cwd] // empty for files with no CWD

		if projectFilter != "" && !strings.Contains(strings.ToLower(name), projectFilter) {
			continue
		}

		for i := range r.records {
			r.records[i].Project = name
		}
		for i := range r.sessions {
			r.sessions[i].Project = name
		}

		allRecords = append(allRecords, r.records...)
		allSessions = append(allSessions, r.sessions...)
		for _, m := range r.unknown {
			unknownModels[m] = true
		}
	}

	slices.SortFunc(allRecords, func(a, b Record) int {
		return a.Time.Compare(b.Time)
	})

	slices.SortFunc(allSessions, func(a, b Session) int {
		return cmp.Compare(a.Date, b.Date)
	})

	slices.Sort(fileErrors)
	var warnings []string
	warnings = append(warnings, fileErrors...)
	for m := range unknownModels {
		warnings = append(warnings, "unknown model: "+m)
	}
	slices.Sort(warnings[len(fileErrors):])

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
	return t.Local(), true
}

// dayBounds tracks min/max timestamps for a single day.
type dayBounds struct {
	min, max time.Time
}

func parseFile(path string, opts Options, isMain bool) ([]Record, []Session, []string, string, error) { //nolint:gocritic // unnamedResult: 5 returns is intentional for this internal function
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("opening log file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// First pass: collect entries, deduplicate by message.id (keep max output_tokens).
	// Also track min/max timestamps per day for session duration (main files only).
	best := map[string]Entry{}
	var fullCWD string
	days := map[string]*dayBounds{} // date string → bounds

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}

		// Extract project from any entry with CWD.
		if fullCWD == "" && e.CWD != "" {
			fullCWD = filepath.Clean(e.CWD)
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
	if err := scanner.Err(); err != nil {
		return nil, nil, nil, "", fmt.Errorf("reading %s: %w", path, err)
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
			Input:      e.Message.Usage.InputTokens,
			Output:     e.Message.Usage.OutputTokens,
			CacheWrite: e.Message.Usage.CacheCreationInputTokens,
			CacheRead:  e.Message.Usage.CacheReadInputTokens,
		})
	}

	// Build per-day sessions for main files.
	var sessions []Session
	if isMain && fullCWD != "" {
		for date, b := range days {
			day, _ := time.ParseInLocation("2006-01-02", date, time.Local)
			if !opts.Since.IsZero() && day.Before(opts.Since) {
				continue
			}
			if !opts.Until.IsZero() && day.After(opts.Until) {
				continue
			}
			sessions = append(sessions, Session{
				Date:     date,
				Duration: b.max.Sub(b.min),
			})
		}
	}

	warnings := slices.Collect(maps.Keys(unknownModels))

	return records, sessions, warnings, fullCWD, nil
}

// disambiguateProjects resolves collisions where multiple CWDs share the same
// filepath.Base() name. For unique base names, the base name is used. For
// collisions, parent path components are added until names are unique.
func disambiguateProjects(cwdsByBase map[string]map[string]bool) map[string]string {
	result := make(map[string]string)

	for base, cwds := range cwdsByBase {
		if len(cwds) == 1 {
			for cwd := range cwds {
				result[cwd] = base
			}
			continue
		}

		// Collision: progressively add parent components until unique.
		cwdList := slices.Collect(maps.Keys(cwds))

		for depth := 2; depth <= 20; depth++ {
			names := make(map[string][]string) // candidate name → CWDs
			for _, cwd := range cwdList {
				name := lastNComponents(cwd, depth)
				names[name] = append(names[name], cwd)
			}

			allUnique := true
			for _, group := range names {
				if len(group) > 1 {
					allUnique = false
					break
				}
			}

			if allUnique {
				for name, group := range names {
					result[group[0]] = name
				}
				break
			}

			if depth == 20 {
				for _, cwd := range cwdList {
					result[cwd] = cwd
				}
			}
		}
	}

	return result
}

// lastNComponents returns the last n path components joined with "/".
func lastNComponents(p string, n int) string {
	p = filepath.Clean(p)
	parts := strings.Split(p, string(filepath.Separator))
	if n >= len(parts) {
		return strings.Join(parts, "/")
	}
	return strings.Join(parts[len(parts)-n:], "/")
}
