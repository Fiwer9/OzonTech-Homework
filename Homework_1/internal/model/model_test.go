package model_test

import (
	"errors"
	"testing"

	"awesomeProject/internal/model"
)

func TestParseID(t *testing.T) {
	id, err := model.ParseID("12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 12 {
		t.Fatalf("got %d", id)
	}

	if _, err := model.ParseID("0"); err == nil {
		t.Fatal("expected error for 0")
	}
	if _, err := model.ParseID("abc"); err == nil {
		t.Fatal("expected error for abc")
	}
}

func TestParseStatus(t *testing.T) {
	for _, want := range []model.Status{
		model.StatusPlanned,
		model.StatusInProgress,
		model.StatusCanceled,
		model.StatusDone,
	} {
		st, err := model.ParseStatus(string(want))
		if err != nil || st != want {
			t.Fatalf("status %s: got %s %v", want, st, err)
		}
	}
	if _, err := model.ParseStatus("unknown"); !errors.Is(err, model.ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestStatusIsClosed(t *testing.T) {
	if !model.StatusDone.IsClosed() || !model.StatusCanceled.IsClosed() {
		t.Fatal("done and canceled must be closed")
	}
	if model.StatusPlanned.IsClosed() || model.StatusInProgress.IsClosed() {
		t.Fatal("planned and in_progress must not be closed")
	}
}
