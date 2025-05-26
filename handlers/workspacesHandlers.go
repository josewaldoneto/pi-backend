package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"projeto-integrador/database"
	"projeto-integrador/models"
	"projeto-integrador/utilities"
	"time"
)

func CreateWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	utilities.LogDebug("Iniciando criação de novo workspace")

	uid := r.Context().Value("userUID")
	if uid == nil {
		utilities.LogError(fmt.Errorf("UID não encontrado no contexto"), "Falha na autenticação")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Decodifica o JSON recebido
	var workspace models.Workspace
	if err := json.NewDecoder(r.Body).Decode(&workspace); err != nil {
		utilities.LogError(err, "Erro ao decodificar JSON do workspace")
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validações básicas
	if workspace.Name == "" {
		utilities.LogError(fmt.Errorf("nome do workspace não fornecido"), "Validação falhou")
		http.Error(w, "Workspace name is required", http.StatusBadRequest)
		return
	}

	utilities.LogDebug("Conectando ao banco de dados para criar workspace")
	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "Erro ao conectar ao banco de dados")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	utilities.LogDebug("Inserindo novo workspace no banco de dados")
	query := `
		INSERT INTO workspaces (name, description, is_public, owner_uid, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`

	var createdAt time.Time
	err = db.QueryRow(
		query,
		workspace.Name,
		workspace.Description,
		workspace.IsPublic,
		uid,
	).Scan(&workspace.ID, &createdAt)

	if err != nil {
		utilities.LogError(err, "Erro ao criar workspace no banco de dados")
		http.Error(w, "Database error while creating workspace", http.StatusInternalServerError)
		return
	}

	workspace.OwnerUID = uid.(string)
	workspace.CreatedAt = createdAt
	workspace.Members = 1 // Criador é o primeiro membro

	utilities.LogDebug("Adicionando criador como admin do workspace")
	_, err = db.Exec(`
		INSERT INTO user_workspace (workspace_id, user_id, role, joined_at)
		VALUES ($1, $2, 'admin', NOW())
	`, workspace.ID, uid)

	if err != nil {
		utilities.LogError(err, "Erro ao adicionar usuário ao workspace")
		http.Error(w, "Database error while adding user to workspace", http.StatusInternalServerError)
		return
	}

	utilities.LogInfo("Workspace criado com sucesso: %s (ID: %d)", workspace.Name, workspace.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(workspace)
}
