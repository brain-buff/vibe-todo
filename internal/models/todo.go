package models

import "time"

// Priority — приоритет задачи: 1 низкий, 2 средний, 3 высокий.
type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 2
	PriorityHigh   Priority = 3
)

type Subtask struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type Todo struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Done        bool       `json:"done"`
	Priority    Priority   `json:"priority"`
	DueAt       *time.Time `json:"due_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Subtasks    []Subtask  `json:"subtasks"`
}

// TodoInput — тело запроса на создание/обновление задачи.
type TodoInput struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    Priority   `json:"priority"`
	DueAt       *time.Time `json:"due_at"`
}

type SubtaskInput struct {
	Title string `json:"title"`
}
