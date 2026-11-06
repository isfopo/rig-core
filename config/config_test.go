package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type sample struct {
	BPM      float64 `toml:"bpm"`
	Name     string  `toml:"name"`
	Channels int     `toml:"channels"`
}

func (s sample) Validate() error {
	if s.BPM <= 0 {
		return errors.New("bpm must be positive")
	}
	if s.Name == "" {
		return errors.New("name required")
	}
	return nil
}

func TestPath(t *testing.T) {
	p, err := Path("looper")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "looper.toml" {
		t.Fatalf("path base = %s", p)
	}
	if _, err := Path(""); err == nil {
		t.Fatal("empty app should fail")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.toml")
	in := sample{BPM: 120.5, Name: "looper", Channels: 2}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	var out sample
	if err := Load(path, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("round trip mismatch: %+v != %+v", out, in)
	}
}

func TestLoadMissing(t *testing.T) {
	var out sample
	err := Load(filepath.Join(t.TempDir(), "missing.toml"), &out)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want os.ErrNotExist", err)
	}
}

func TestLoadApp(t *testing.T) {
	// Path resolution only; loading a missing conventional file reports not-exist.
	_, err := LoadApp("definitely-not-an-app", &sample{})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want os.ErrNotExist", err)
	}
}

func TestLoadValidated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := Save(path, sample{BPM: 0, Name: ""}); err != nil {
		t.Fatal(err)
	}
	var out sample
	if err := LoadValidated(path, &out); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCheckHelpers(t *testing.T) {
	if err := CheckNonEmpty("name", "x"); err != nil {
		t.Fatal(err)
	}
	if err := CheckNonEmpty("name", ""); err == nil {
		t.Fatal("expected error")
	}
	if err := CheckPositive("bpm", 1); err != nil {
		t.Fatal(err)
	}
	if err := CheckPositive("bpm", 0); err == nil {
		t.Fatal("expected error")
	}
	if err := CheckRange("x", 5, 1, 10); err != nil {
		t.Fatal(err)
	}
	if err := CheckRange("x", 0, 1, 10); err == nil {
		t.Fatal("expected error")
	}
}

func TestSaveCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "app.toml")
	if err := Save(path, sample{BPM: 1, Name: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
