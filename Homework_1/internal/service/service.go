package service

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"awesomeProject/internal/model"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
)

type Service struct {
	store Store
	now   func() time.Time
}

func New(store Store) *Service {
	return NewWithClock(store, time.Now)
}

func NewWithClock(store Store, now func() time.Time) *Service {
	return &Service{store: store, now: now}
}

type ListFilter struct {
	Status  *model.Status
	Overdue bool
	Query   string
	Page    int
	Limit   int
}

type ListResult struct {
	Tasks []model.Task
	Total int
	Page  int
	Limit int
	Pages int
}

type IncomingTask struct {
	Title    string
	Deadline time.Time
}

type ImportError struct {
	Index int
	Err   error
}

type ImportReport struct {
	Added  []model.Task
	Errors []ImportError
}

func (s *Service) Add(title string, deadline time.Time) (model.Task, error) {
	title = strings.TrimSpace(title)
	if err := s.validateTitle(title); err != nil {
		return model.Task{}, fmt.Errorf("добавление задачи: %w", err)
	}
	if err := s.validateDeadline(deadline); err != nil {
		return model.Task{}, fmt.Errorf("добавление задачи: %w", err)
	}

	task := model.Task{
		Title:    title,
		Status:   model.StatusPlanned,
		Deadline: truncateDay(deadline),
	}
	created, err := s.store.Create(task)
	if err != nil {
		return model.Task{}, fmt.Errorf("добавление задачи: %w", err)
	}
	return created, nil
}

func (s *Service) Get(id model.ID) (model.Task, error) {
	task, err := s.store.Get(id)
	if err != nil {
		return model.Task{}, fmt.Errorf("получение задачи %d: %w", id, err)
	}
	return task, nil
}

func (s *Service) List(filter ListFilter) (ListResult, error) {
	page := filter.Page
	limit := filter.Limit
	if page == 0 {
		page = DefaultPage
	}
	if limit == 0 {
		limit = DefaultLimit
	}
	if page < 1 {
		return ListResult{}, fmt.Errorf("номер страницы должен быть >= 1")
	}
	if limit < 1 {
		return ListResult{}, fmt.Errorf("размер страницы должен быть >= 1")
	}

	all, err := s.store.List()
	if err != nil {
		return ListResult{}, fmt.Errorf("список задач: %w", err)
	}

	query := strings.ToLower(strings.TrimSpace(filter.Query))
	today := s.today()
	filtered := make([]model.Task, 0, len(all))
	for _, t := range all {
		if filter.Status != nil && t.Status != *filter.Status {
			continue
		}
		if filter.Overdue {
			if !t.Deadline.Before(today) || t.Status.IsClosed() {
				continue
			}
		}
		if query != "" && !strings.Contains(strings.ToLower(t.Title), query) {
			continue
		}
		filtered = append(filtered, t)
	}

	slices.SortFunc(filtered, func(a, b model.Task) int {
		if a.Deadline.Before(b.Deadline) {
			return -1
		}
		if a.Deadline.After(b.Deadline) {
			return 1
		}
		return int(a.ID) - int(b.ID)
	})

	total := len(filtered)
	pages := 0
	if total > 0 {
		pages = (total + limit - 1) / limit
	}

	start := (page - 1) * limit
	var pageItems []model.Task
	if start < total {
		end := start + limit
		if end > total {
			end = total
		}
		pageItems = filtered[start:end]
	}

	return ListResult{
		Tasks: pageItems,
		Total: total,
		Page:  page,
		Limit: limit,
		Pages: pages,
	}, nil
}

func (s *Service) SetStatus(id model.ID, status model.Status) (model.Task, error) {
	task, err := s.store.Get(id)
	if err != nil {
		return model.Task{}, fmt.Errorf("смена статуса задачи %d: %w", id, err)
	}
	if task.Status.IsClosed() {
		return model.Task{}, fmt.Errorf("смена статуса задачи %d: %w", id, model.ErrClosed)
	}
	if task.Status == status {
		return model.Task{}, fmt.Errorf("смена статуса задачи %d: %w", id, model.ErrSameStatus)
	}
	task.Status = status
	if err := s.store.Update(task); err != nil {
		return model.Task{}, fmt.Errorf("смена статуса задачи %d: %w", id, err)
	}
	return task, nil
}

func (s *Service) Edit(id model.ID, title *string, deadline *time.Time) (model.Task, error) {
	if title == nil && deadline == nil {
		return model.Task{}, fmt.Errorf("редактирование задачи %d: %w", id, model.ErrNothingToUpdate)
	}

	task, err := s.store.Get(id)
	if err != nil {
		return model.Task{}, fmt.Errorf("редактирование задачи %d: %w", id, err)
	}
	if task.Status.IsClosed() {
		return model.Task{}, fmt.Errorf("редактирование задачи %d: %w", id, model.ErrClosed)
	}

	if title != nil {
		t := strings.TrimSpace(*title)
		if err := s.validateTitle(t); err != nil {
			return model.Task{}, fmt.Errorf("редактирование задачи %d: %w", id, err)
		}
		task.Title = t
	}
	if deadline != nil {
		if err := s.validateDeadline(*deadline); err != nil {
			return model.Task{}, fmt.Errorf("редактирование задачи %d: %w", id, err)
		}
		task.Deadline = truncateDay(*deadline)
	}

	if err := s.store.Update(task); err != nil {
		return model.Task{}, fmt.Errorf("редактирование задачи %d: %w", id, err)
	}
	return task, nil
}

func (s *Service) Delete(id model.ID) error {
	if err := s.store.Delete(id); err != nil {
		return fmt.Errorf("удаление задачи %d: %w", id, err)
	}
	return nil
}

func (s *Service) Import(items []IncomingTask) ImportReport {
	report := ImportReport{
		Added:  make([]model.Task, 0),
		Errors: make([]ImportError, 0),
	}
	for i, item := range items {
		idx := i + 1
		created, err := s.Add(item.Title, item.Deadline)
		if err != nil {
			report.Errors = append(report.Errors, ImportError{Index: idx, Err: err})
			continue
		}
		report.Added = append(report.Added, created)
	}
	return report
}

func (s *Service) validateTitle(title string) error {
	if title == "" {
		return model.ErrEmptyTitle
	}
	return nil
}

func (s *Service) validateDeadline(deadline time.Time) error {
	if deadline.IsZero() {
		return model.ErrDeadlineInPast
	}
	if truncateDay(deadline).Before(s.today()) {
		return model.ErrDeadlineInPast
	}
	return nil
}

func (s *Service) today() time.Time {
	return truncateDay(s.now())
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
