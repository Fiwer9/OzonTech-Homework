package service

import "awesomeProject/internal/model"

type Store interface {
	Create(task model.Task) (model.Task, error)
	Get(id model.ID) (model.Task, error)
	Update(task model.Task) error
	Delete(id model.ID) error
	List() ([]model.Task, error)
}
