package model

import "errors"

var (
	ErrNotFound        = errors.New("задача не найдена")
	ErrEmptyTitle      = errors.New("заголовок не может быть пустым")
	ErrDeadlineInPast  = errors.New("дедлайн не может быть в прошлом")
	ErrInvalidStatus   = errors.New("неизвестный статус")
	ErrSameStatus      = errors.New("задача уже находится в этом статусе")
	ErrClosed          = errors.New("задачу в статусе done или canceled нельзя изменять")
	ErrNothingToUpdate = errors.New("нужно указать новый заголовок и/или дедлайн")
)
