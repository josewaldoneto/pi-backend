package models

import "time"

type Note struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	WorkspaceID int64      `json:"workspace_id"`
	Status      string     `json:"status"` // active, completed, archived
	UserID      string     `json:"user_id"`
	Type        string     `json:"type"` // note, event, reminder, file
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
