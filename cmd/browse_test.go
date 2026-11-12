package cmd

import (
	"bytes"
	"testing"
)

func TestBrowsePrintsRepositoryLocationWithoutBrowser(t *testing.T) {
	command := newBrowseCommand()
	command.SetArgs([]string{"matcxz/codeberg-cli", "issues/12", "--no-browser"})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.Flags().Lookup("no-browser").Value.Set("true")

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEBERG_HOST", "https://codeberg.org")
	var output bytes.Buffer
	command.SetOut(&output)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "https://codeberg.org/matcxz/codeberg-cli/issues/12\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestBrowseRejectsUnsafeLocations(t *testing.T) {
	for _, location := range []string{"../other", "issues/../../other", "issues/1?token=x", "#fragment"} {
		command := newBrowseCommand()
		command.SetArgs([]string{"matcxz/codeberg-cli", location, "--no-browser"})
		command.SetOut(&bytes.Buffer{})
		command.SetErr(&bytes.Buffer{})
		if err := command.Execute(); err == nil {
			t.Errorf("browse location %q succeeded, want error", location)
		}
	}
}
