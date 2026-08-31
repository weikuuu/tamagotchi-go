package pet

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	now := time.Date(2026, 8, 29, 12, 30, 0, 0, time.UTC)
	want := State{Hunger: 42, Energy: 17, Happiness: 88, LastTick: now}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Hunger != want.Hunger || got.Energy != want.Energy || got.Happiness != want.Happiness {
		t.Errorf("stats mismatch: got=%+v want=%+v", got, want)
	}
	if !got.LastTick.Equal(want.LastTick) {
		t.Errorf("LastTick = %v, want %v", got.LastTick, want.LastTick)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")

	_, err := Load(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load() error = %v, want os.ErrNotExist", err)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garbage.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error for invalid JSON")
	}
}
