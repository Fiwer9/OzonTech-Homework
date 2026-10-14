package cli

import "testing"

func TestSplitCommand(t *testing.T) {
	args := splitCommand(`add 2026-12-01 "Купить молоко"`)
	if len(args) != 3 || args[0] != "add" || args[1] != "2026-12-01" || args[2] != "Купить молоко" {
		t.Fatalf("%q", args)
	}
}

func TestParseFlags(t *testing.T) {
	f, err := parseFlags([]string{"--status", "done", "--overdue", "--query", "молоко", "--page", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if f.values["status"] != "done" || f.values["query"] != "молоко" || f.values["page"] != "2" {
		t.Fatalf("%+v", f.values)
	}
	if !f.bools["overdue"] {
		t.Fatal("overdue")
	}
}

func TestParseFlagsUnexpected(t *testing.T) {
	if _, err := parseFlags([]string{"oops"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseFlagsMissingValue(t *testing.T) {
	if _, err := parseFlags([]string{"--status"}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := parseFlags([]string{"--query", "--overdue"}); err == nil {
		t.Fatal("expected error for query without value")
	}
}

func TestSplitCommandEmpty(t *testing.T) {
	if args := splitCommand("   "); len(args) != 0 {
		t.Fatalf("%q", args)
	}
}
