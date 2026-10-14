package service_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"awesomeProject/internal/model"
	"awesomeProject/internal/service"
	"awesomeProject/internal/storage"
)

func fixedNow() time.Time {
	return time.Date(2026, 10, 14, 12, 0, 0, 0, time.Local)
}

func newService() *service.Service {
	return service.NewWithClock(storage.NewMemoryStore(), fixedNow)
}

func date(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

func TestAddValidation(t *testing.T) {
	svc := newService()

	if _, err := svc.Add("  ", date("2026-12-01")); !errors.Is(err, model.ErrEmptyTitle) {
		t.Fatalf("empty title: %v", err)
	}
	if _, err := svc.Add("задача", date("2026-10-13")); !errors.Is(err, model.ErrDeadlineInPast) {
		t.Fatalf("past deadline: %v", err)
	}

	task, err := svc.Add("  купить молоко  ", date("2026-10-14"))
	if err != nil {
		t.Fatalf("today should be allowed: %v", err)
	}
	if task.ID != 1 || task.Status != model.StatusPlanned || task.Title != "купить молоко" {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func TestIDsAreNotReused(t *testing.T) {
	svc := newService()
	a, _ := svc.Add("a", date("2026-12-01"))
	b, _ := svc.Add("b", date("2026-12-01"))
	if err := svc.Delete(a.ID); err != nil {
		t.Fatal(err)
	}
	c, err := svc.Add("c", date("2026-12-01"))
	if err != nil {
		t.Fatal(err)
	}
	if c.ID != 3 {
		t.Fatalf("expected id 3 after deleting 1, got %d (b was %d)", c.ID, b.ID)
	}
	if _, err := svc.Get(a.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("deleted task still available: %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	svc := newService()
	_, err := svc.Get(99)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestListSortAndFilters(t *testing.T) {
	svc := newService()
	_, _ = svc.Add("поздняя", date("2026-12-20"))
	first, _ := svc.Add("ранняя", date("2026-11-01"))
	_, _ = svc.Add("ранняя-2", date("2026-11-01"))

	res, err := svc.List(service.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tasks) != 3 {
		t.Fatalf("want 3, got %d", len(res.Tasks))
	}
	if res.Tasks[0].Title != "ранняя" || res.Tasks[1].Title != "ранняя-2" || res.Tasks[2].Title != "поздняя" {
		t.Fatalf("bad order: %+v %+v %+v", res.Tasks[0], res.Tasks[1], res.Tasks[2])
	}

	st := model.StatusPlanned
	res, _ = svc.List(service.ListFilter{Status: &st})
	if res.Total != 3 {
		t.Fatalf("status filter: %d", res.Total)
	}

	res, _ = svc.List(service.ListFilter{Query: "РАННЯЯ"})
	if res.Total != 2 {
		t.Fatalf("query filter: %d", res.Total)
	}

	_, _ = svc.SetStatus(first.ID, model.StatusInProgress)
	overdueDeadline := date("2026-10-14")
	past, err := svc.Add("ещё нет просрочки сегодня", overdueDeadline)
	if err != nil {
		t.Fatal(err)
	}
	_ = past
	res, _ = svc.List(service.ListFilter{Overdue: true})
	if res.Total != 0 {
		t.Fatalf("today is not overdue, got %d", res.Total)
	}
}

func TestListOverdue(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := service.NewWithClock(store, func() time.Time {
		return time.Date(2026, 10, 10, 0, 0, 0, 0, time.Local)
	})
	open, _ := svc.Add("открытая просрочка", date("2026-10-10"))
	done, _ := svc.Add("закрытая", date("2026-10-10"))
	_, _ = svc.SetStatus(done.ID, model.StatusDone)

	later := service.NewWithClock(store, func() time.Time {
		return time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local)
	})
	res, err := later.List(service.ListFilter{Overdue: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Tasks[0].ID != open.ID {
		t.Fatalf("overdue: %+v", res)
	}
}

func TestListPagination(t *testing.T) {
	svc := newService()
	for i := 0; i < 5; i++ {
		_, err := svc.Add("t", date("2026-12-01"))
		if err != nil {
			t.Fatal(err)
		}
	}
	res, err := svc.List(service.ListFilter{Page: 2, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 5 || res.Pages != 3 || len(res.Tasks) != 2 {
		t.Fatalf("pagination: %+v len=%d", res, len(res.Tasks))
	}
	if res.Tasks[0].ID != 3 || res.Tasks[1].ID != 4 {
		t.Fatalf("page items: %+v", res.Tasks)
	}

	res, _ = svc.List(service.ListFilter{})
	if res.Page != 1 || res.Limit != 20 {
		t.Fatalf("defaults: page=%d limit=%d", res.Page, res.Limit)
	}
}

func TestSetStatusRules(t *testing.T) {
	svc := newService()
	task, _ := svc.Add("a", date("2026-12-01"))

	if _, err := svc.SetStatus(task.ID, model.StatusPlanned); !errors.Is(err, model.ErrSameStatus) {
		t.Fatalf("same status: %v", err)
	}
	updated, err := svc.SetStatus(task.ID, model.StatusInProgress)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != model.StatusInProgress {
		t.Fatal(updated.Status)
	}
	_, _ = svc.SetStatus(task.ID, model.StatusDone)
	if _, err := svc.SetStatus(task.ID, model.StatusPlanned); !errors.Is(err, model.ErrClosed) {
		t.Fatalf("closed: %v", err)
	}
}

func TestEditRules(t *testing.T) {
	svc := newService()
	task, _ := svc.Add("старое", date("2026-12-01"))

	if _, err := svc.Edit(task.ID, nil, nil); !errors.Is(err, model.ErrNothingToUpdate) {
		t.Fatalf("nothing: %v", err)
	}
	empty := "  "
	if _, err := svc.Edit(task.ID, &empty, nil); !errors.Is(err, model.ErrEmptyTitle) {
		t.Fatalf("empty title: %v", err)
	}
	past := date("2026-01-01")
	if _, err := svc.Edit(task.ID, nil, &past); !errors.Is(err, model.ErrDeadlineInPast) {
		t.Fatalf("past: %v", err)
	}

	title := "новое"
	next := date("2026-12-15")
	updated, err := svc.Edit(task.ID, &title, &next)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "новое" || updated.Deadline.Format("2006-01-02") != "2026-12-15" {
		t.Fatalf("%+v", updated)
	}

	_, _ = svc.SetStatus(task.ID, model.StatusCanceled)
	if _, err := svc.Edit(task.ID, &title, nil); !errors.Is(err, model.ErrClosed) {
		t.Fatalf("edit closed: %v", err)
	}
}

func TestImportPartialSuccess(t *testing.T) {
	svc := newService()
	report := svc.Import([]service.IncomingTask{
		{Title: "ok", Deadline: date("2026-12-01")},
		{Title: "", Deadline: date("2026-12-01")},
		{Title: "past", Deadline: date("2020-01-01")},
		{Title: "ok2", Deadline: date("2026-12-02")},
	})
	if len(report.Added) != 2 {
		t.Fatalf("added: %+v", report.Added)
	}
	if len(report.Errors) != 2 {
		t.Fatalf("errors: %+v", report.Errors)
	}
	if report.Errors[0].Index != 2 || report.Errors[1].Index != 3 {
		t.Fatalf("indexes: %+v", report.Errors)
	}
}

func TestListStableOrder(t *testing.T) {
	svc := newService()
	_, _ = svc.Add("b", date("2026-12-02"))
	_, _ = svc.Add("a", date("2026-12-01"))
	first, _ := svc.List(service.ListFilter{})
	second, _ := svc.List(service.ListFilter{})
	if len(first.Tasks) != len(second.Tasks) {
		t.Fatal("len")
	}
	for i := range first.Tasks {
		if first.Tasks[i].ID != second.Tasks[i].ID {
			t.Fatalf("order changed: %+v vs %+v", first.Tasks, second.Tasks)
		}
	}
}

func TestListStatusFilterExcludesOthers(t *testing.T) {
	svc := newService()
	a, _ := svc.Add("planned", date("2026-12-01"))
	b, _ := svc.Add("progress", date("2026-12-02"))
	_, _ = svc.SetStatus(b.ID, model.StatusInProgress)

	st := model.StatusInProgress
	res, err := svc.List(service.ListFilter{Status: &st})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Tasks[0].ID != b.ID {
		t.Fatalf("status filter: %+v (a=%d)", res, a.ID)
	}
}

func TestListOverdueExcludesCanceled(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := service.NewWithClock(store, func() time.Time {
		return time.Date(2026, 10, 10, 0, 0, 0, 0, time.Local)
	})
	open, _ := svc.Add("open", date("2026-10-10"))
	canceled, _ := svc.Add("canceled", date("2026-10-10"))
	_, _ = svc.SetStatus(canceled.ID, model.StatusCanceled)

	later := service.NewWithClock(store, func() time.Time {
		return time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local)
	})
	res, err := later.List(service.ListFilter{Overdue: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Tasks[0].ID != open.ID {
		t.Fatalf("overdue must skip canceled: %+v", res)
	}
}

func TestListPaginationDefaultsAndErrors(t *testing.T) {
	svc := newService()
	_, _ = svc.Add("t", date("2026-12-01"))

	if _, err := svc.List(service.ListFilter{Page: -1}); err == nil {
		t.Fatal("page -1")
	}
	if _, err := svc.List(service.ListFilter{Limit: -5}); err == nil {
		t.Fatal("limit -5")
	}

	res, err := svc.List(service.ListFilter{Page: 10, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Pages != 1 || len(res.Tasks) != 0 {
		t.Fatalf("empty page: %+v", res)
	}
}

func TestListPaginationUsesFilteredTotal(t *testing.T) {
	svc := newService()
	_, _ = svc.Add("alpha", date("2026-12-01"))
	_, _ = svc.Add("beta", date("2026-12-01"))
	_, _ = svc.Add("alpha-2", date("2026-12-01"))

	res, err := svc.List(service.ListFilter{Query: "alpha", Page: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 || res.Pages != 2 || len(res.Tasks) != 1 {
		t.Fatalf("filtered pagination: %+v", res)
	}
}

func TestSetStatusCanceledCannotChange(t *testing.T) {
	svc := newService()
	task, _ := svc.Add("a", date("2026-12-01"))
	_, _ = svc.SetStatus(task.ID, model.StatusCanceled)
	if _, err := svc.SetStatus(task.ID, model.StatusPlanned); !errors.Is(err, model.ErrClosed) {
		t.Fatalf("canceled: %v", err)
	}
}

func TestNotFoundOnMutations(t *testing.T) {
	svc := newService()
	title := "x"
	if _, err := svc.SetStatus(7, model.StatusInProgress); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("status: %v", err)
	}
	if _, err := svc.Edit(7, &title, nil); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("edit: %v", err)
	}
	if err := svc.Delete(7); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("delete: %v", err)
	}
}

func TestEditTitleOnlyKeepsDeadline(t *testing.T) {
	svc := newService()
	task, _ := svc.Add("old", date("2026-12-01"))
	title := "new"
	updated, err := svc.Edit(task.ID, &title, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "new" || updated.Deadline.Format("2006-01-02") != "2026-12-01" {
		t.Fatalf("%+v", updated)
	}
}

func TestGetReturnsTask(t *testing.T) {
	svc := newService()
	created, _ := svc.Add("one", date("2026-12-01"))
	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID || got.Title != "one" || got.Status != model.StatusPlanned {
		t.Fatalf("%+v", got)
	}
}

func TestNewUsesRealClock(t *testing.T) {
	svc := service.New(storage.NewMemoryStore())
	task, err := svc.Add("future", date("2099-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != model.StatusPlanned {
		t.Fatal(task.Status)
	}
}

func TestFileStoreThroughServicePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	store := storage.NewFileStore(path)
	svc := service.NewWithClock(store, fixedNow)

	created, err := svc.Add("disk", date("2026-12-01"))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(created.ID); err != nil {
		t.Fatal(err)
	}
	again, err := svc.Add("after-delete", date("2026-12-02"))
	if err != nil {
		t.Fatal(err)
	}
	if again.ID == created.ID {
		t.Fatal("id reused after delete on disk")
	}

	reloaded := service.NewWithClock(storage.NewFileStore(path), fixedNow)
	got, err := reloaded.Get(again.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "after-delete" {
		t.Fatalf("%+v", got)
	}
	if _, err := reloaded.Get(created.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("deleted still on disk: %v", err)
	}
}
