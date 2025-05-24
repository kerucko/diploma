package models

import "time"

type Task struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Solution    string    `json:"solution"`
	CreatedAt   time.Time `json:"created_at"`
	SolvedAt    time.Time `json:"solved_at,omitempty"`
}
