package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadConfigUsesPrivatePermissions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := Config{
		Host:     "https://codeberg.org",
		Token:    "secret-token",
		Username: "matcxz",
	}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}

	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("config permissions = %04o, want 0600", info.Mode().Perm())
	}
	if info.Name() != filepath.Base(path) {
		t.Fatalf("config filename = %q, want %q", info.Name(), filepath.Base(path))
	}
}

func TestLoadMissingConfigUsesDefaultHost(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != (Config{Host: DefaultHost}) {
		t.Fatalf("Load() = %#v, want default host", got)
	}
}

func TestLoadRejectsConfigReadableByOtherUsers(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"host":%q,"token":"secret"}`, DefaultHost)), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Fatalf("Load() error = %v, want permissions guidance", err)
	}
}
