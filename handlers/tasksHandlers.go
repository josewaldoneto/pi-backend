package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"projeto-integrador/models"

	"github.com/gorilla/mux"
)

// createTaskHandler cria uma nova tarefa em um workspace
func createTaskHandler(w http.ResponseWriter, r *http.Request) {
	LogDebug("Iniciando criação de nova tarefa")

	vars := mux.Vars(r)
	workspaceID := vars["id"]

	// Obter UID do usuário a partir do token Firebase
	uid, err := getUIDFromToken(r)
	if err != nil {
		LogError(err, "Falha na autenticação ao criar tarefa")
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}

	// Verificar se o usuário é membro do workspace
	isMember, err := isWorkspaceMember(uid, workspaceID)
	if err != nil || !isMember {
		LogError(err, "Usuário não tem permissão para criar tarefa no workspace")
		http.Error(w, "Acesso não autorizado ao workspace", http.StatusForbidden)
		return
	}

	var task struct {
		Title      string    `json:"title"`
		Content    string    `json:"content"`
		Priority   string    `json:"priority"`
		Status     string    `json:"status"`
		Expiration time.Time `json:"expiration"`
	}

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		LogError(err, "Erro ao decodificar JSON da tarefa")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validar prioridade
	validPriorities := map[string]bool{"low": true, "medium": true, "high": true}
	if !validPriorities[task.Priority] {
		LogError(fmt.Errorf("prioridade inválida: %s", task.Priority), "Validação falhou")
		http.Error(w, "Prioridade inválida", http.StatusBadRequest)
		return
	}

	// Validar status
	validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true}
	if task.Status != "" && !validStatuses[task.Status] {
		LogError(fmt.Errorf("status inválido: %s", task.Status), "Validação falhou")
		http.Error(w, "Status inválido", http.StatusBadRequest)
		return
	}

	// Obter ID do usuário
	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE firebase_uid = $1", uid).Scan(&userID)
	if err != nil {
		LogError(err, "Erro ao obter ID do usuário")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	LogDebug("Inserindo nova tarefa no banco de dados")
	query := `INSERT INTO tarefas (title, conteudo, prioridade, status, expiracao, criado_por, workspace_id)
              VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	var id int
	err = db.QueryRow(query,
		task.Title,
		task.Content,
		task.Priority,
		task.Status,
		task.Expiration,
		userID,
		workspaceID,
	).Scan(&id)
	if err != nil {
		LogError(err, "Erro ao inserir tarefa no banco de dados")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	LogInfo("Tarefa criada com sucesso: %s (ID: %d)", task.Title, id)
	response := map[string]int{"id": id}
	json.NewEncoder(w).Encode(response)
}

// getTasksHandler lista todas as tarefas de um workspace
func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	LogDebug("Iniciando listagem de tarefas")

	vars := mux.Vars(r)
	workspaceID := vars["id"]

	// Obter UID do usuário a partir do token Firebase
	uid, err := getUIDFromToken(r)
	if err != nil {
		LogError(err, "Falha na autenticação ao listar tarefas")
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}

	// Verificar se o usuário é membro do workspace
	isMember, err := isWorkspaceMember(uid, workspaceID)
	if err != nil || !isMember {
		LogError(err, "Usuário não tem permissão para listar tarefas do workspace")
		http.Error(w, "Acesso não autorizado ao workspace", http.StatusForbidden)
		return
	}

	// Obter parâmetros de query para filtragem
	queryParams := r.URL.Query()
	statusFilter := queryParams.Get("status")
	priorityFilter := queryParams.Get("priority")

	LogDebug("Buscando tarefas com filtros - status: %s, prioridade: %s", statusFilter, priorityFilter)

	// Construir query base
	query := `
        SELECT t.id, t.title, t.conteudo, t.prioridade, t.status, t.expiracao, 
               t.criado_por, t.workspace_id, t.created_at, u.username
        FROM tarefas t
        JOIN users u ON t.criado_por = u.id
        WHERE t.workspace_id = $1
    `
	params := []interface{}{workspaceID}
	paramCount := 2

	// Adicionar filtros
	if statusFilter != "" {
		query += fmt.Sprintf(" AND t.status = $%d", paramCount)
		params = append(params, statusFilter)
		paramCount++
	}

	if priorityFilter != "" {
		query += fmt.Sprintf(" AND t.prioridade = $%d", paramCount)
		params = append(params, priorityFilter)
		paramCount++
	}

	query += " ORDER BY t.created_at DESC"

	rows, err := db.Query(query, params...)
	if err != nil {
		LogError(err, "Erro ao buscar tarefas no banco de dados")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := []map[string]interface{}{}
	for rows.Next() {
		var task models.Task
		var createdByUsername string
		var expiration sql.NullTime

		err := rows.Scan(
			&task.ID, &task.Title, &task.Content, &task.Priority, &task.Status,
			&expiration, &task.CreatedBy, &task.WorkspaceID, &task.CreatedAt,
			&createdByUsername,
		)
		if err != nil {
			LogError(err, "Erro ao ler resultado da query de tarefas")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		taskMap := map[string]interface{}{
			"id":           task.ID,
			"title":        task.Title,
			"content":      task.Content,
			"priority":     task.Priority,
			"status":       task.Status,
			"created_by":   createdByUsername,
			"workspace_id": task.WorkspaceID,
			"created_at":   task.CreatedAt,
		}

		if expiration.Valid {
			taskMap["expiration"] = expiration.Time
		}

		tasks = append(tasks, taskMap)
	}

	LogInfo("Tarefas listadas com sucesso - total: %d", len(tasks))
	json.NewEncoder(w).Encode(tasks)
}

// updateTaskHandler atualiza uma tarefa existente
func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	LogDebug("Iniciando atualização de tarefa")

	vars := mux.Vars(r)
	taskID := vars["id"]

	// Obter UID do usuário a partir do token Firebase
	uid, err := getUIDFromToken(r)
	if err != nil {
		LogError(err, "Falha na autenticação ao atualizar tarefa")
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}

	// Verificar se o usuário é o criador da tarefa ou membro admin do workspace
	canEdit, err := canEditTask(uid, taskID)
	if err != nil || !canEdit {
		LogError(err, "Usuário não tem permissão para editar a tarefa")
		http.Error(w, "Sem permissão para editar esta tarefa", http.StatusForbidden)
		return
	}

	var updates struct {
		Title      *string    `json:"title"`
		Content    *string    `json:"content"`
		Priority   *string    `json:"priority"`
		Status     *string    `json:"status"`
		Expiration *time.Time `json:"expiration"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		LogError(err, "Erro ao decodificar JSON de atualização")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	LogDebug("Construindo query de atualização para tarefa %s", taskID)
	// Construir query dinâmica
	query := "UPDATE tarefas SET "
	params := []interface{}{}
	paramCount := 1

	if updates.Title != nil {
		query += fmt.Sprintf("title = $%d, ", paramCount)
		params = append(params, *updates.Title)
		paramCount++
	}

	if updates.Content != nil {
		query += fmt.Sprintf("conteudo = $%d, ", paramCount)
		params = append(params, *updates.Content)
		paramCount++
	}

	if updates.Priority != nil {
		// Validar prioridade
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true}
		if !validPriorities[*updates.Priority] {
			LogError(fmt.Errorf("prioridade inválida: %s", *updates.Priority), "Validação falhou")
			http.Error(w, "Prioridade inválida", http.StatusBadRequest)
			return
		}
		query += fmt.Sprintf("prioridade = $%d, ", paramCount)
		params = append(params, *updates.Priority)
		paramCount++
	}

	if updates.Status != nil {
		// Validar status
		validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true}
		if !validStatuses[*updates.Status] {
			LogError(fmt.Errorf("status inválido: %s", *updates.Status), "Validação falhou")
			http.Error(w, "Status inválido", http.StatusBadRequest)
			return
		}
		query += fmt.Sprintf("status = $%d, ", paramCount)
		params = append(params, *updates.Status)
		paramCount++
	}

	if updates.Expiration != nil {
		query += fmt.Sprintf("expiracao = $%d, ", paramCount)
		params = append(params, *updates.Expiration)
		paramCount++
	}

	// Remover a vírgula final e adicionar a cláusula WHERE
	query = strings.TrimSuffix(query, ", ") + " WHERE id = $" + strconv.Itoa(paramCount)
	params = append(params, taskID)

	_, err = db.Exec(query, params...)
	if err != nil {
		LogError(err, "Erro ao atualizar tarefa no banco de dados")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	LogInfo("Tarefa atualizada com sucesso: %s", taskID)
	w.WriteHeader(http.StatusNoContent)
}

// deleteTaskHandler remove uma tarefa
func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	LogDebug("Iniciando exclusão de tarefa")

	vars := mux.Vars(r)
	taskID := vars["id"]

	// Obter UID do usuário a partir do token Firebase
	uid, err := getUIDFromToken(r)
	if err != nil {
		LogError(err, "Falha na autenticação ao excluir tarefa")
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}

	// Verificar se o usuário é o criador da tarefa ou admin do workspace
	canDelete, err := canDeleteTask(uid, taskID)
	if err != nil || !canDelete {
		LogError(err, "Usuário não tem permissão para excluir a tarefa")
		http.Error(w, "Sem permissão para deletar esta tarefa", http.StatusForbidden)
		return
	}

	_, err = db.Exec("DELETE FROM tarefas WHERE id = $1", taskID)
	if err != nil {
		LogError(err, "Erro ao excluir tarefa do banco de dados")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	LogInfo("Tarefa excluída com sucesso: %s", taskID)
	w.WriteHeader(http.StatusNoContent)
}

// Funções auxiliares para tarefas
func canEditTask(uid string, taskID string) (bool, error) {
	LogDebug("Verificando permissão de edição para usuário %s na tarefa %s", uid, taskID)

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE firebase_uid = $1", uid).Scan(&userID)
	if err != nil {
		LogError(err, "Erro ao obter ID do usuário para verificação de permissão")
		return false, err
	}

	// Verificar se é o criador da tarefa
	var isCreator bool
	err = db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM tarefas WHERE id = $1 AND criado_por = $2)",
		taskID, userID,
	).Scan(&isCreator)
	if err != nil {
		LogError(err, "Erro ao verificar se usuário é criador da tarefa")
		return false, err
	}
	if isCreator {
		LogDebug("Usuário é o criador da tarefa")
		return true, nil
	}

	// Verificar se é admin do workspace da tarefa
	var isAdmin bool
	err = db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM membros_workspace mw
            JOIN tarefas t ON mw.workspace_id = t.workspace_id
            WHERE t.id = $1 AND mw.usuario_id = $2 AND mw.role = 'admin'
        )`, taskID, userID,
	).Scan(&isAdmin)

	if err != nil {
		LogError(err, "Erro ao verificar se usuário é admin do workspace")
		return false, err
	}

	if isAdmin {
		LogDebug("Usuário é admin do workspace")
	} else {
		LogDebug("Usuário não tem permissão de edição")
	}

	return isAdmin, nil
}

func canDeleteTask(uid string, taskID string) (bool, error) {
	LogDebug("Verificando permissão de exclusão para usuário %s na tarefa %s", uid, taskID)
	// Mesma lógica que canEditTask para este exemplo
	return canEditTask(uid, taskID)
}
