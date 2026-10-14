package storage

import (
	"sync"

	"awesomeProject/internal/model"
)

type MemoryStore struct {
	mu     sync.Mutex
	nextID int
	tasks  map[model.ID]model.Task
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nextID: 1,
		tasks:  make(map[model.ID]model.Task),
	}
}

func (s *MemoryStore) Create(task model.Task) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task.ID = model.ID(s.nextID)
	s.nextID++
	s.tasks[task.ID] = task
	return task, nil
}

func (s *MemoryStore) Get(id model.ID) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return model.Task{}, model.ErrNotFound
	}
	return task, nil
}

func (s *MemoryStore) Update(task model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[task.ID]; !ok {
		return model.ErrNotFound
	}
	s.tasks[task.ID] = task
	return nil
}

func (s *MemoryStore) Delete(id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return model.ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}

func (s *MemoryStore) List() ([]model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]model.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	return out, nil
}
