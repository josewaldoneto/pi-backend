package models

import (
	"time"
)

// TarefaStub representa o registro minimalista de uma tarefa no PostgreSQL.
// Os detalhes completos da tarefa (título, descrição, status, etc.)
// residirão primariamente no Firestore.
type TarefaStub struct {
	ID             int64     `json:"id"`               // ID interno auto-incrementado do PostgreSQL
	FirestoreDocID string    `json:"firestore_doc_id"` // ID do documento correspondente no Firestore
	WorkspaceID    int64     `json:"workspace_id"`     // Chave estrangeira para workspaces.id
	CriadoPor      int64     `json:"criado_por"`       // Chave estrangeira para users.id (o ID numérico do criador)
	CreatedAt      time.Time `json:"created_at"`       // Data de criação do registro no PostgreSQL
	UpdatedAt      time.Time `json:"updated_at"`       // Data da última atualização do registro no PostgreSQL
}

type TaskAttachment struct {
	Filename string `json:"filename" firestore:"filename"`
	URL      string `json:"url" firestore:"url"`
	Filetype string `json:"filetype" firestore:"filetype,omitempty"`
}

// TaskDetailsFirestore representa os detalhes de uma tarefa armazenados no Firestore.
type TaskDetailsFirestore struct {
	Title          string     `json:"title" firestore:"title"`
	Description    string     `json:"description" firestore:"description,omitempty"`
	Status         string     `json:"status" firestore:"status"`               // ex: "pending", "in_progress", "completed"
	Priority       string     `json:"priority" firestore:"priority,omitempty"` // ex: "low", "medium", "high"
	ExpirationDate *time.Time `json:"expiration_date,omitempty" firestore:"expiration_date,omitempty"`
	Attachment     string     `json:"attachment,omitempty" firestore:"attachment,omitempty"`

	WorkspaceIDPg      int64     `json:"-" firestore:"workspace_id_pg"` // ID do workspace no PostgreSQL
	CreatorFirebaseUID string    `json:"creator_firebase_uid" firestore:"creator_firebase_uid"`
	CreatedAt          time.Time `json:"created_at" firestore:"created_at"`           // Idealmente um firestore.ServerTimestamp na escrita
	LastUpdatedAt      time.Time `json:"last_updated_at" firestore:"last_updated_at"` // Idealmente um firestore.ServerTimestamp na escrita/atualização
}

// Para escrita, você pode querer uma struct de input que não inclua campos gerados pelo servidor como CreatedAt
type CreateTaskInput struct {
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	Priority       string     `json:"priority"`
	ExpirationDate *time.Time `json:"expiration_date"`

	Attachment string `json:"attachment"`
}

type UpdateTaskInput struct {
	Title          *string    `json:"title"` // Ponteiros para indicar quais campos atualizar
	Description    *string    `json:"description"`
	Status         *string    `json:"status"`
	Priority       *string    `json:"priority"`
	ExpirationDate *time.Time `json:"expiration_date"`
	Attachment     *string    `json:"attachment"` // Para atualizar ou remover, pode ser complexo
}
