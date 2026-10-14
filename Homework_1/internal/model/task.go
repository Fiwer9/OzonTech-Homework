package model

import "time"

type Task struct {
	ID       ID
	Title    string
	Status   Status
	Deadline time.Time
}
