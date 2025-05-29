package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"projeto-integrador/database"
	"projeto-integrador/firebase"
	"projeto-integrador/models"
	"projeto-integrador/utilities"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore" // Para firestore.ServerTimestamp se usado
	"github.com/google/uuid"        // Para gerar IDs únicos para o Firestore
	"github.com/gorilla/mux"
)

const tasksSubCollection = "tasks" // Nome da subcoleção de tarefas no Firestore

// CreateTaskHandler cria uma nova tarefa dentro de um workspace
func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceIDStr, ok := vars["workspace_id"]
	if !ok {
		utilities.LogError(fmt.Errorf("workspace_id não encontrado"), "CreateTaskHandler: Parâmetro ausente")
		http.Error(w, "Workspace ID is required", http.StatusBadRequest)
		return
	}
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		utilities.LogError(err, "CreateTaskHandler: workspace_id inválido")
		http.Error(w, "Invalid Workspace ID format", http.StatusBadRequest)
		return
	}

	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := context.Background()

	var input models.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utilities.LogError(err, "CreateTaskHandler: Erro ao decodificar JSON")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if input.Title == "" {
		http.Error(w, "Task title is required", http.StatusBadRequest)
		return
	}
	if input.Status == "" { // Definir um status padrão se não fornecido
		input.Status = "pending"
	}

	// Conectar ao PostgreSQL
	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "CreateTaskHandler: Erro ao conectar ao PG")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Autorização: Verificar se o usuário é membro do workspace
	isMember, err := models.IsWorkspaceMember(db, requestingUserFirebaseUID, workspaceID)
	if err != nil {
		utilities.LogError(err, "CreateTaskHandler: Erro ao verificar membresia")
		http.Error(w, "Failed to verify workspace membership", http.StatusInternalServerError)
		return
	}
	if !isMember {
		utilities.LogInfo(fmt.Sprintf("CreateTaskHandler: Usuário %s não autorizado no workspace %d", requestingUserFirebaseUID, workspaceID))
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Obter o users.id (inteiro) do criador para o stub PG
	var creatorUserIDPg int64
	err = db.QueryRow("SELECT id FROM users WHERE firebase_uid = $1", requestingUserFirebaseUID).Scan(&creatorUserIDPg)
	if err != nil {
		utilities.LogError(err, "CreateTaskHandler: Usuário criador não encontrado no PG")
		http.Error(w, "Authenticated user not found in database", http.StatusInternalServerError)
		return
	}

	// 1. Criar documento no Firestore
	firestoreDocID := uuid.New().String() // Gerar um ID único para o documento Firestore
	taskDetails := models.TaskDetailsFirestore{
		Title:              input.Title,
		Description:        input.Description,
		Status:             input.Status,
		Priority:           input.Priority,
		ExpirationDate:     input.ExpirationDate,
		Attachment:         input.Attachment, // Assume que o cliente já fez upload e está enviando metadados
		WorkspaceIDPg:      workspaceID,
		CreatorFirebaseUID: requestingUserFirebaseUID,
		CreatedAt:          time.Now(), // Ou use firestore.ServerTimestamp se a struct suportar
		LastUpdatedAt:      time.Now(), // Ou use firestore.ServerTimestamp
	}

	// O caminho no Firestore será /workspaces/{postgres_workspace_id}/tasks/{firestore_task_doc_id}
	// Nota: {postgres_workspace_id} precisa ser uma string no caminho do Firestore.
	workspaceDocIDForFirestore := strconv.FormatInt(workspaceID, 10)
	firestoreClient, err := firebase.GetFirestoreClient() // Função que retorna o cliente Firestore
	if err != nil {
		utilities.LogError(err, "CreateTaskHandler: Erro ao obter cliente Firestore")
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}
	_, err = firestoreClient.Collection("workspaces").Doc(workspaceDocIDForFirestore).Collection(tasksSubCollection).Doc(firestoreDocID).Set(ctx, taskDetails)
	if err != nil {
		utilities.LogError(err, "CreateTaskHandler: Erro ao criar tarefa no Firestore")
		http.Error(w, "Failed to create task details", http.StatusInternalServerError)
		return
	}

	// 2. Criar stub no PostgreSQL
	taskStub := models.TarefaStub{
		FirestoreDocID: firestoreDocID,
		WorkspaceID:    workspaceID,
		CriadoPor:      creatorUserIDPg,
		// CreatedAt e UpdatedAt terão default no PG
	}
	_, err = db.Exec(`INSERT INTO tarefas (firestore_doc_id, workspace_id, criado_por) VALUES ($1, $2, $3)`,
		taskStub.FirestoreDocID, taskStub.WorkspaceID, taskStub.CriadoPor)
	if err != nil {
		utilities.LogError(err, "CreateTaskHandler: Erro ao criar stub da tarefa no PG")
		// Tentar reverter a criação no Firestore (lógica de compensação)
		_, delErr := firestoreClient.Collection("workspaces").Doc(workspaceDocIDForFirestore).Collection(tasksSubCollection).Doc(firestoreDocID).Delete(ctx)
		if delErr != nil {
			utilities.LogError(delErr, "CreateTaskHandler: FALHA AO REVERTER criação no Firestore após erro no PG")
		}
		http.Error(w, "Failed to create task record", http.StatusInternalServerError)
		return
	}

	// Retornar os detalhes completos da tarefa (do Firestore) ou apenas uma confirmação
	// Para consistência, vamos buscar do Firestore o que foi salvo
	createdTaskDoc, err := firestoreClient.Collection("workspaces").Doc(workspaceDocIDForFirestore).Collection(tasksSubCollection).Doc(firestoreDocID).Get(ctx)
	if err != nil {
		utilities.LogError(err, "CreateTaskHandler: Erro ao buscar tarefa recém-criada do Firestore para resposta")
		// Ainda assim, a criação foi um sucesso, então podemos retornar 201 sem o corpo completo
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Task created successfully", "firestoreDocId": firestoreDocID})
		return
	}

	var finalTaskData models.TaskDetailsFirestore
	createdTaskDoc.DataTo(&finalTaskData)

	utilities.LogInfo(fmt.Sprintf("CreateTaskHandler: Tarefa %s criada no workspace %d", firestoreDocID, workspaceID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(finalTaskData)
}

// ListTasksHandler lista todas as tarefas de um workspace (buscando do Firestore)
func ListTasksHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceIDStr, ok := vars["workspace_id"]
	if !ok {
		http.Error(w, "Workspace ID is required", http.StatusBadRequest)
		return
	}
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Workspace ID format", http.StatusBadRequest)
		return
	}

	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := context.Background()

	db, err := database.ConnectPostgres() // Necessário para checar membresia
	if err != nil {
		utilities.LogError(err, "ListTasksHandler: Erro ao conectar ao PG")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	isMember, err := models.IsWorkspaceMember(db, requestingUserFirebaseUID, workspaceID)
	if err != nil || !isMember {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	firestoreClient, err := firebase.GetFirestoreClient() // Função que retorna o cliente Firestore
	if err != nil {
		utilities.LogError(err, "ListTasksHandler: Erro ao obter cliente Firestore")
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	workspaceDocIDForFirestore := strconv.FormatInt(workspaceID, 10)
	iter := firestoreClient.Collection("workspaces").Doc(workspaceDocIDForFirestore).Collection(tasksSubCollection).Documents(ctx)
	defer iter.Stop()

	var tasks []map[string]interface{} // Usando map para flexibilidade ou defina uma struct de resposta
	for {
		doc, err := iter.Next()
		if err != nil {
			// TODO: Melhorar tratamento de erro do iter.Next() (ex: iterator.Done)
			if err.Error() == "EOF" || strings.Contains(err.Error(), "iterator done") { // Adaptar para o erro correto de fim de iteração
				break
			}
			utilities.LogError(err, "ListTasksHandler: Erro ao iterar tarefas do Firestore")
			http.Error(w, "Failed to retrieve tasks", http.StatusInternalServerError)
			return
		}
		taskData := doc.Data()
		taskData["id"] = doc.Ref.ID // Adiciona o ID do documento Firestore aos dados
		tasks = append(tasks, taskData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GetTaskHandler busca os detalhes de uma tarefa específica do Firestore
func GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceIDStr, _ := vars["workspace_id"]
	taskDocID, ok := vars["task_doc_id"]
	if !ok {
		http.Error(w, "Task Document ID is required", http.StatusBadRequest)
		return
	}
	workspaceID, _ := strconv.ParseInt(workspaceIDStr, 10, 64) // Erro já tratado em Listar, mas bom ter aqui também

	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := context.Background()

	db, err := database.ConnectPostgres()
	if err != nil { /* ... erro PG ... */
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	isMember, err := models.IsWorkspaceMember(db, requestingUserFirebaseUID, workspaceID)
	if err != nil || !isMember {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	firestoreClient, err := firebase.GetFirestoreClient() // Função que retorna o cliente Firestore
	if err != nil {
		utilities.LogError(err, "GetTaskHandler: Erro ao obter cliente Firestore")
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	workspaceDocIDForFirestore := strconv.FormatInt(workspaceID, 10)
	docSnap, err := firestoreClient.Collection("workspaces").Doc(workspaceDocIDForFirestore).Collection(tasksSubCollection).Doc(taskDocID).Get(ctx)
	if err != nil {
		// TODO: Verificar se erro é 'document not found'
		utilities.LogError(err, "GetTaskHandler: Erro ao buscar tarefa do Firestore")
		http.Error(w, "Task not found or error fetching", http.StatusNotFound) // Ou 500
		return
	}

	var taskData models.TaskDetailsFirestore
	if err := docSnap.DataTo(&taskData); err != nil {
		utilities.LogError(err, "GetTaskHandler: Erro ao converter dados da tarefa")
		http.Error(w, "Error processing task data", http.StatusInternalServerError)
		return
	}

	// Adicionar o ID do documento à resposta se não estiver na struct
	// Se TaskDetailsFirestore não tiver um campo ID, podemos retornar um map ou uma struct de resposta:
	response := map[string]interface{}{
		"id":                 taskDocID,
		"title":              taskData.Title,
		"description":        taskData.Description,
		"status":             taskData.Status,
		"priority":           taskData.Priority,
		"expirationDate":     taskData.ExpirationDate,
		"attachment":         taskData.Attachment,
		"creatorFirebaseUid": taskData.CreatorFirebaseUID,
		"createdAt":          taskData.CreatedAt,
		"lastUpdatedAt":      taskData.LastUpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateTaskHandler atualiza uma tarefa existente no Firestore e o stub no PG
func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceIDStr, _ := vars["workspace_id"]
	taskDocID, ok := vars["task_doc_id"]
	if !ok { /* ... erro ... */
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}
	workspaceID, _ := strconv.ParseInt(workspaceIDStr, 10, 64)

	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := context.Background()

	var input models.UpdateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	db, err := database.ConnectPostgres()
	if err != nil { /* ... erro PG ... */
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	isMember, err := models.IsWorkspaceMember(db, requestingUserFirebaseUID, workspaceID)
	if err != nil || !isMember { // Adicionar verificação de permissão de edição (ex: criador, admin)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Construir lista de updates para o Firestore
	// Nota: firestore.Update requer []firestore.Update. Ex: {Path: "title", Value: "Novo Título"}
	var updates []firestore.Update
	if input.Title != nil {
		updates = append(updates, firestore.Update{Path: "title", Value: *input.Title})
	}
	if input.Description != nil {
		updates = append(updates, firestore.Update{Path: "description", Value: *input.Description})
	}
	if input.Status != nil {
		updates = append(updates, firestore.Update{Path: "status", Value: *input.Status})
	}
	if input.Priority != nil {
		updates = append(updates, firestore.Update{Path: "priority", Value: *input.Priority})
	}
	if input.ExpirationDate != nil { // Se for para permitir desmarcar, precisa de lógica especial
		updates = append(updates, firestore.Update{Path: "expiration_date", Value: input.ExpirationDate})
	}
	if input.Attachment != nil { // Atualizar anexo é mais complexo (deletar antigo do storage?)
		updates = append(updates, firestore.Update{Path: "attachment", Value: input.Attachment})
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}
	// Adicionar campos de auditoria
	updates = append(updates, firestore.Update{Path: "last_updated_by_firebase_uid", Value: requestingUserFirebaseUID})
	updates = append(updates, firestore.Update{Path: "last_updated_at", Value: firestore.ServerTimestamp})

	firestoreClient, err := firebase.GetFirestoreClient() // Função que retorna o cliente Firestore
	if err != nil {
		utilities.LogError(err, "UpdateTaskHandler: Erro ao obter cliente Firestore")
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	workspaceDocIDForFirestore := strconv.FormatInt(workspaceID, 10)
	taskRef := firestoreClient.Collection("workspaces").Doc(workspaceDocIDForFirestore).Collection(tasksSubCollection).Doc(taskDocID)
	_, err = taskRef.Update(ctx, updates)
	if err != nil {
		utilities.LogError(err, "UpdateTaskHandler: Erro ao atualizar tarefa no Firestore")
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	// Atualizar o updated_at no stub do PostgreSQL
	_, err = db.Exec("UPDATE tarefas SET updated_at = NOW() WHERE firestore_doc_id = $1 AND workspace_id = $2", taskDocID, workspaceID)
	if err != nil {
		utilities.LogError(err, "UpdateTaskHandler: Erro ao atualizar timestamp do stub da tarefa no PG")
		// A atualização no Firestore foi bem-sucedida, mas o stub PG não. Logar, mas não necessariamente reverter.
	}

	utilities.LogInfo(fmt.Sprintf("UpdateTaskHandler: Tarefa %s atualizada no workspace %d", taskDocID, workspaceID))
	w.WriteHeader(http.StatusOK) // Ou retornar o documento atualizado
	json.NewEncoder(w).Encode(map[string]string{"message": "Task updated successfully"})
}

// DeleteTaskHandler deleta uma tarefa (do Firestore e o stub do PG)
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceIDStr, _ := vars["workspace_id"]
	taskDocID, ok := vars["task_doc_id"]
	if !ok { /* ... erro ... */
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}
	workspaceID, _ := strconv.ParseInt(workspaceIDStr, 10, 64)

	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := context.Background()

	db, err := database.ConnectPostgres()
	if err != nil { /* ... erro PG ... */
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	isMember, err := models.IsWorkspaceMember(db, requestingUserFirebaseUID, workspaceID)
	if err != nil || !isMember { // Adicionar verificação de permissão de deleção (ex: criador, admin)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	firestoreClient, err := firebase.GetFirestoreClient() // Função que retorna o cliente Firestore

	if err != nil {
		utilities.LogError(err, "DeleteTaskHandler: Erro ao obter cliente Firestore")
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	// 1. Deletar do Firestore
	workspaceDocIDForFirestore := strconv.FormatInt(workspaceID, 10)
	_, err = firestoreClient.Collection("workspaces").Doc(workspaceDocIDForFirestore).Collection(tasksSubCollection).Doc(taskDocID).Delete(ctx)
	if err != nil {
		// TODO: Verificar se o erro é "not found" - nesse caso, a tarefa já pode ter sido deletada.
		utilities.LogError(err, "DeleteTaskHandler: Erro ao deletar tarefa do Firestore")
		http.Error(w, "Failed to delete task from primary store", http.StatusInternalServerError)
		return
	}

	// 2. Deletar stub do PostgreSQL
	result, err := db.Exec("DELETE FROM tarefas WHERE firestore_doc_id = $1 AND workspace_id = $2", taskDocID, workspaceID)
	if err != nil {
		utilities.LogError(err, "DeleteTaskHandler: Erro ao deletar stub da tarefa do PG")
		// Firestore foi deletado, mas PG não. Logar para reconciliação manual.
		http.Error(w, "Task deleted from primary store, but failed to delete record", http.StatusInternalServerError) // Ou 200 com aviso
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utilities.LogInfo(fmt.Sprintf("DeleteTaskHandler: Stub da tarefa %s não encontrado no PG para workspace %d (ou já deletado)", taskDocID, workspaceID))
		// Isso pode ser OK se o Firestore foi a fonte principal da deleção.
	}

	utilities.LogInfo(fmt.Sprintf("DeleteTaskHandler: Tarefa %s deletada do workspace %d", taskDocID, workspaceID))
	w.WriteHeader(http.StatusNoContent)
}
