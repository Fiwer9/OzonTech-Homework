package model

import (
	"fmt"
	"strconv"
)

type ID int

func ParseID(s string) (ID, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("некорректный идентификатор %q", s)
	}
	return ID(n), nil
}

type Status string

const (
	StatusPlanned    Status = "planned"
	StatusInProgress Status = "in_progress"
	StatusCanceled   Status = "canceled"
	StatusDone       Status = "done"
)

func ParseStatus(s string) (Status, error) {
	st := Status(s)
	switch st {
	case StatusPlanned, StatusInProgress, StatusCanceled, StatusDone:
		return st, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidStatus, s)
	}
}

func (s Status) IsClosed() bool {
	return s == StatusDone || s == StatusCanceled
}
