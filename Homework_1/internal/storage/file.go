package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"awesomeProject/internal/model"
)

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

type snapshot struct {
	NextID int             `json:"next_id"`
	Tasks  []persistedTask `json:"tasks"`
}

type persistedTask struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Deadline string `json:"deadline"`
}

func (s *FileStore) Create(task model.Task) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap, err := s.load()
	if err != nil {
		return model.Task{}, fmt.Errorf("файловое хранилище: создание: %w", err)
	}
	task.ID = model.ID(snap.NextID)
	snap.NextID++
	snap.Tasks = append(snap.Tasks, toPersisted(task))
	if err := s.save(snap); err != nil {
		return model.Task{}, fmt.Errorf("файловое хранилище: создание: %w", err)
	}
	return task, nil
}

func (s *FileStore) Get(id model.ID) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap, err := s.load()
	if err != nil {
		return model.Task{}, fmt.Errorf("файловое хранилище: чтение: %w", err)
	}
	for _, t := range snap.Tasks {
		if t.ID == int(id) {
			task, err := fromPersisted(t)
			if err != nil {
				return model.Task{}, fmt.Errorf("файловое хранилище: чтение: %w", err)
			}
			return task, nil
		}
	}
	return model.Task{}, model.ErrNotFound
}

func (s *FileStore) Update(task model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap, err := s.load()
	if err != nil {
		return fmt.Errorf("файловое хранилище: обновление: %w", err)
	}
	for i, t := range snap.Tasks {
		if t.ID == int(task.ID) {
			snap.Tasks[i] = toPersisted(task)
			if err := s.save(snap); err != nil {
				return fmt.Errorf("файловое хранилище: обновление: %w", err)
			}
			return nil
		}
	}
	return model.ErrNotFound
}

func (s *FileStore) Delete(id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap, err := s.load()
	if err != nil {
		return fmt.Errorf("файловое хранилище: удаление: %w", err)
	}
	found := false
	kept := make([]persistedTask, 0, len(snap.Tasks))
	for _, t := range snap.Tasks {
		if t.ID == int(id) {
			found = true
			continue
		}
		kept = append(kept, t)
	}
	if !found {
		return model.ErrNotFound
	}
	snap.Tasks = kept
	if err := s.save(snap); err != nil {
		return fmt.Errorf("файловое хранилище: удаление: %w", err)
	}
	return nil
}

func (s *FileStore) List() ([]model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap, err := s.load()
	if err != nil {
		return nil, fmt.Errorf("файловое хранилище: список: %w", err)
	}
	out := make([]model.Task, 0, len(snap.Tasks))
	for _, t := range snap.Tasks {
		task, err := fromPersisted(t)
		if err != nil {
			return nil, fmt.Errorf("файловое хранилище: список: %w", err)
		}
		out = append(out, task)
	}
	return out, nil
}

func (s *FileStore) load() (snapshot, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return snapshot{NextID: 1, Tasks: []persistedTask{}}, nil
		}
		return snapshot{}, fmt.Errorf("чтение файла %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return snapshot{NextID: 1, Tasks: []persistedTask{}}, nil
	}

	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return snapshot{}, fmt.Errorf("разбор файла %s: %w", s.path, err)
	}
	if snap.Tasks == nil {
		snap.Tasks = []persistedTask{}
	}
	if snap.NextID <= 0 {
		maxID := 0
		for _, t := range snap.Tasks {
			if t.ID > maxID {
				maxID = t.ID
			}
		}
		snap.NextID = maxID + 1
	}
	return snap, nil
}

func (s *FileStore) save(snap snapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("сериализация: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("запись файла %s: %w", s.path, err)
	}
	return nil
}

func toPersisted(t model.Task) persistedTask {
	return persistedTask{
		ID:       int(t.ID),
		Title:    t.Title,
		Status:   string(t.Status),
		Deadline: t.Deadline.Format("2006-01-02"),
	}
}

func fromPersisted(t persistedTask) (model.Task, error) {
	deadline, err := time.ParseInLocation("2006-01-02", t.Deadline, time.Local)
	if err != nil {
		return model.Task{}, fmt.Errorf("дедлайн задачи %d: %w", t.ID, err)
	}
	status, err := model.ParseStatus(t.Status)
	if err != nil {
		return model.Task{}, fmt.Errorf("статус задачи %d: %w", t.ID, err)
	}
	return model.Task{
		ID:       model.ID(t.ID),
		Title:    t.Title,
		Status:   status,
		Deadline: deadline,
	}, nil
}
