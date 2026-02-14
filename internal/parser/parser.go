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

// Parse reads all JSONL files under ~/.claude/projects and returns deduplicated records.
func Parse(opts Options) ([]Record, []string, error) {
	dir, err := claudeDir()
	if err != nil {
		return nil, nil, err
	}
	return parseDir(dir, opts)
}

func parseDir(dir string, opts Options) ([]Record, []string, error) {
	// Main session files: <project>/<uuid>.jsonl
	mainPattern := filepath.Join(dir, "*", "*.jsonl")
	// Subagent files: <project>/<uuid>/subagents/agent-*.jsonl
	subPattern := filepath.Join(dir, "*", "*", "subagents", "*.jsonl")

	mainFiles, err := filepath.Glob(mainPattern)
	if err != nil {
		return nil, nil, err
	}
	subFiles, _ := filepath.Glob(subPattern)
	files := append(mainFiles, subFiles...)

	var allRecords []Record
	unknownModels := map[string]bool{}

	for _, f := range files {
		records, unknown, err := parseFile(f, opts)
		if err != nil {
			continue // skip broken files
		}
		allRecords = append(allRecords, records...)
		for _, m := range unknown {
			unknownModels[m] = true
		}
	}

	sort.Slice(allRecords, func(i, j int) bool {
		return allRecords[i].Time.Before(allRecords[j].Time)
	})

	var warnings []string
	for m := range unknownModels {
		warnings = append(warnings, "unknown model: "+m)
	}
	sort.Strings(warnings)

	return allRecords, warnings, nil
}

func parseFile(path string, opts Options) ([]Record, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	// First pass: collect entries, deduplicate by message.id (keep max output_tokens).
	best := map[string]Entry{}
	var project string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.Type != "assistant" || e.Message.ID == "" {
			continue
		}
		if project == "" && e.CWD != "" {
			project = filepath.Base(e.CWD)
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
		return nil, nil, nil
	}

	var records []Record
	unknownModels := map[string]bool{}

	for _, e := range best {
		// Skip entries with all-zero usage (e.g. <synthetic>).
		if e.Message.Usage.IsZero() {
			continue
		}

		t, err := time.Parse(time.RFC3339Nano, e.Timestamp)
		if err != nil {
			t, err = time.Parse(time.RFC3339, e.Timestamp)
			if err != nil {
				continue
			}
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

	var warnings []string
	for m := range unknownModels {
		warnings = append(warnings, m)
	}

	return records, warnings, nil
}
