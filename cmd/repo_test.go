package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/matcxz/codeberg-cli/internal/forgejo"
)

func TestPrintRepositoryTableTruncatesToTerminalWidth(t *testing.T) {
	repositories := []forgejo.Repository{{
		FullName:    "matcxz/a-repository-with-a-very-long-name",
		Description: "A very long description that must be shortened to fit the terminal",
		Private:     true,
		Stars:       12,
	}}

	var output bytes.Buffer
	if err := printRepositoryTable(&output, repositories, 50); err != nil {
		t.Fatal(err)
	}

	got := output.String()
	if !strings.Contains(got, "...") {
		t.Fatalf("expected truncated fields to end with ellipses, got:\n%s", got)
	}
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if width := len([]rune(line)); width > 50 {
			t.Errorf("line width = %d, want at most 50: %q", width, line)
		}
	}
}

func TestPrintRepositoryTableUsesCompactColumnsOnNarrowTerminal(t *testing.T) {
	repositories := []forgejo.Repository{{
		FullName: "matcxz/long-repository-name",
		Private:  true,
	}}
	var output bytes.Buffer
	if err := printRepositoryTable(&output, repositories, 20); err != nil {
		t.Fatal(err)
	}

	for _, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
		if width := len([]rune(line)); width > 20 {
			t.Errorf("line width = %d, want at most 20: %q", width, line)
		}
	}
	if !strings.Contains(output.String(), "...") {
		t.Fatalf("expected repository name to be truncated, got:\n%s", output.String())
	}
}

func TestTruncateUsesDotsWhenValueDoesNotFit(t *testing.T) {
	for _, test := range []struct {
		width int
		want  string
	}{
		{width: 8, want: "abcde..."},
		{width: 3, want: "..."},
		{width: 2, want: ".."},
	} {
		if got := truncate("abcdefghi", test.width); got != test.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", "abcdefghi", test.width, got, test.want)
		}
	}
}

func TestPrintNumberedListRespectsNarrowWidth(t *testing.T) {
	var output bytes.Buffer
	entries := [][3]string{{"123456789", "closed", "A long issue title that must fit"}}
	if err := printNumberedListWidth(&output, entries, 25); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
		if width := len([]rune(line)); width > 25 {
			t.Errorf("line width = %d, want at most 25: %q", width, line)
		}
	}
	if !strings.Contains(output.String(), "...") {
		t.Fatalf("expected long title to be truncated: %q", output.String())
	}
}
