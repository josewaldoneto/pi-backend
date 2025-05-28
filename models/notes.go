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
