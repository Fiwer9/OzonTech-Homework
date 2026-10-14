package storage_test

import (
	"errors"
	"testing"
	"time"

	"awesomeProject/internal/model"
	"awesomeProject/internal/storage"
)

func sample(title string) model.Task {
	return model.Task{
		Title:    title,
		Status:   model.StatusPlanned,
		Deadline: time.Date(2026, 12, 1, 0, 0, 0, 0, time.Local),
	}
}

func TestMemoryStoreCRUD(t *testing.T) {
	s := storage.NewMemoryStore()

	a, err := s.Create(sample("a"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(sample("b"))
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 || a.ID == b.ID {
		t.Fatalf("ids: %d %d", a.ID, b.ID)
	}

	got, err := s.Get(a.ID)
	if err != nil || got.Title != "a" {
		t.Fatalf("get: %v %+v", err, got)
	}

	a.Title = "a2"
	a.Status = model.StatusInProgress
	if err := s.Update(a); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Get(a.ID)
	if got.Title != "a2" || got.Status != model.StatusInProgress {
		t.Fatalf("update: %+v", got)
	}

	list, err := s.List()
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	if err := s.Delete(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(a.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("deleted: %v", err)
	}

	c, err := s.Create(sample("c"))
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == a.ID {
		t.Fatal("memory store reused id")
	}
}

func TestMemoryStoreNotFound(t *testing.T) {
	s := storage.NewMemoryStore()
	if _, err := s.Get(1); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("get: %v", err)
	}
	if err := s.Update(model.Task{ID: 1}); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("update: %v", err)
	}
	if err := s.Delete(1); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("delete: %v", err)
	}
}
