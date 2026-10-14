package storage_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"awesomeProject/internal/model"
	"awesomeProject/internal/storage"
)

func TestFileStorePersistsAndKeepsIDs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	s1 := storage.NewFileStore(path)
	created, err := s1.Create(model.Task{
		Title:    "a",
		Status:   model.StatusPlanned,
		Deadline: time.Date(2026, 12, 1, 0, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 1 {
		t.Fatalf("id=%d", created.ID)
	}
	if err := s1.Delete(created.ID); err != nil {
		t.Fatal(err)
	}

	s2 := storage.NewFileStore(path)
	again, err := s2.Create(model.Task{
		Title:    "b",
		Status:   model.StatusPlanned,
		Deadline: time.Date(2026, 12, 2, 0, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != 2 {
		t.Fatalf("id reused: %d", again.ID)
	}

	got, err := s2.Get(again.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "b" {
		t.Fatalf("%+v", got)
	}

	list, err := s2.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
}

func TestFileStoreMissingFile(t *testing.T) {
	s := storage.NewFileStore(filepath.Join(t.TempDir(), "missing.json"))
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("%+v", list)
	}
}

func TestFileStoreUpdateAndNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	s := storage.NewFileStore(path)
	task, err := s.Create(model.Task{
		Title:    "old",
		Status:   model.StatusPlanned,
		Deadline: time.Date(2026, 12, 1, 0, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatal(err)
	}

	task.Title = "new"
	task.Status = model.StatusInProgress
	if err := s.Update(task); err != nil {
		t.Fatal(err)
	}

	reloaded := storage.NewFileStore(path)
	got, err := reloaded.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "new" || got.Status != model.StatusInProgress {
		t.Fatalf("%+v", got)
	}

	if _, err := reloaded.Get(99); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("get: %v", err)
	}
	if err := reloaded.Update(model.Task{ID: 99}); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("update: %v", err)
	}
	if err := reloaded.Delete(99); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("delete: %v", err)
	}
}

func TestFileStoreEmptyAndCorrupt(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(empty, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	s := storage.NewFileStore(empty)
	list, err := s.List()
	if err != nil || len(list) != 0 {
		t.Fatalf("empty: %v %+v", err, list)
	}

	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.NewFileStore(bad).List(); err == nil {
		t.Fatal("expected corrupt json error")
	}
}

func TestFileStoreRecoversNextID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	raw := `{
  "next_id": 0,
  "tasks": [
    {"id": 4, "title": "x", "status": "planned", "deadline": "2026-12-01"}
  ]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	s := storage.NewFileStore(path)
	created, err := s.Create(model.Task{
		Title:    "y",
		Status:   model.StatusPlanned,
		Deadline: time.Date(2026, 12, 2, 0, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 5 {
		t.Fatalf("expected recovered id 5, got %d", created.ID)
	}
}
