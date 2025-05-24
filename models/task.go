package models

import "time"

type Task struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	Expiration  time.Time `json:"expiration"`
	CreatedBy   int       `json:"created_by"`
	WorkspaceID int       `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
}
