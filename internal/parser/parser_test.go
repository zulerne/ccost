package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	projDir := filepath.Join(dir, "test-project-abc")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "session.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestParseBasic(t *testing.T) {
	data := `{"type":"assistant","timestamp":"2026-02-14T10:00:00.000Z","cwd":"/home/user/myproject","message":{"id":"msg_001","model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":200,"cache_read_input_tokens":300}}}
{"type":"user","timestamp":"2026-02-14T10:01:00.000Z","message":{"text":"hello"}}
{"type":"assistant","timestamp":"2026-02-14T10:02:00.000Z","cwd":"/home/user/myproject","message":{"id":"msg_002","model":"claude-opus-4-6","usage":{"input_tokens":150,"output_tokens":75,"cache_creation_input_tokens":0,"cache_read_input_tokens":500}}}
`
	dir := setupTestDir(t, data)
	records, warnings, err := parseDir(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Project != "myproject" {
		t.Errorf("expected project 'myproject', got %q", records[0].Project)
	}
}

func TestDeduplication(t *testing.T) {
	data := `{"type":"assistant","timestamp":"2026-02-14T10:00:00.000Z","cwd":"/home/user/proj","message":{"id":"msg_dup","model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
{"type":"assistant","timestamp":"2026-02-14T10:00:01.000Z","cwd":"/home/user/proj","message":{"id":"msg_dup","model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	dir := setupTestDir(t, data)
	records, _, err := parseDir(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record after dedup, got %d", len(records))
	}
	if records[0].Output != 50 {
		t.Errorf("expected output=50 (max), got %d", records[0].Output)
	}
}

func TestFilterSince(t *testing.T) {
	data := `{"type":"assistant","timestamp":"2026-02-10T10:00:00.000Z","cwd":"/home/user/proj","message":{"id":"msg_old","model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
{"type":"assistant","timestamp":"2026-02-14T10:00:00.000Z","cwd":"/home/user/proj","message":{"id":"msg_new","model":"claude-opus-4-6","usage":{"input_tokens":200,"output_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	dir := setupTestDir(t, data)
	since, _ := time.Parse("2006-01-02", "2026-02-12")
	records, _, err := parseDir(dir, Options{Since: since})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record after since filter, got %d", len(records))
	}
	if records[0].Input != 200 {
		t.Errorf("expected input=200, got %d", records[0].Input)
	}
}

func TestFilterProject(t *testing.T) {
	data := `{"type":"assistant","timestamp":"2026-02-14T10:00:00.000Z","cwd":"/home/user/myapp","message":{"id":"msg_001","model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	dir := setupTestDir(t, data)

	records, _, _ := parseDir(dir, Options{Project: "myapp"})
	if len(records) != 1 {
		t.Errorf("expected 1 record matching 'myapp', got %d", len(records))
	}

	records, _, _ = parseDir(dir, Options{Project: "other"})
	if len(records) != 0 {
		t.Errorf("expected 0 records for 'other', got %d", len(records))
	}
}

func TestUnknownModel(t *testing.T) {
	data := `{"type":"assistant","timestamp":"2026-02-14T10:00:00.000Z","cwd":"/home/user/proj","message":{"id":"msg_001","model":"claude-future-99","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	dir := setupTestDir(t, data)
	_, warnings, err := parseDir(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestZeroTokensSkipped(t *testing.T) {
	data := `{"type":"assistant","timestamp":"2026-02-14T10:00:00.000Z","cwd":"/home/user/proj","message":{"id":"msg_syn","model":"<synthetic>","usage":{"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
{"type":"assistant","timestamp":"2026-02-14T10:01:00.000Z","cwd":"/home/user/proj","message":{"id":"msg_real","model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	dir := setupTestDir(t, data)
	records, warnings, err := parseDir(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record (synthetic skipped), got %d", len(records))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestSubagentFiles(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "test-project-abc")

	// Main session file
	mainData := `{"type":"assistant","timestamp":"2026-02-14T10:00:00.000Z","cwd":"/home/user/proj","message":{"id":"msg_main","model":"claude-opus-4-6","usage":{"input_tokens":1000,"output_tokens":500,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "session-abc.jsonl"), []byte(mainData), 0o644); err != nil {
		t.Fatal(err)
	}

	// Subagent file
	subDir := filepath.Join(projDir, "session-abc", "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subData := `{"type":"assistant","timestamp":"2026-02-14T10:05:00.000Z","cwd":"/home/user/proj","message":{"id":"msg_sub","model":"claude-haiku-4-5-20251001","usage":{"input_tokens":50,"output_tokens":20,"cache_creation_input_tokens":100,"cache_read_input_tokens":200}}}
`
	if err := os.WriteFile(filepath.Join(subDir, "agent-a123.jsonl"), []byte(subData), 0o644); err != nil {
		t.Fatal(err)
	}

	records, warnings, err := parseDir(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records (main + subagent), got %d", len(records))
	}

	// Check that subagent haiku was found
	hasHaiku := false
	for _, r := range records {
		if r.Model == "claude-haiku-4-5" {
			hasHaiku = true
		}
	}
	if !hasHaiku {
		t.Error("expected subagent with haiku model to be parsed")
	}
}

func TestModelNormalization(t *testing.T) {
	data := `{"type":"assistant","timestamp":"2026-02-14T10:00:00.000Z","cwd":"/home/user/proj","message":{"id":"msg_001","model":"claude-sonnet-4-5-20250929","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	dir := setupTestDir(t, data)
	records, warnings, err := parseDir(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Model != "claude-sonnet-4-5" {
		t.Errorf("expected normalized model 'claude-sonnet-4-5', got %q", records[0].Model)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for known model, got %v", warnings)
	}
}
