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
	"time" // Usado no CreateWorkspaceHandler

	"github.com/gorilla/mux" // Assumindo o uso do gorilla/mux para roteamento
)

func CreateWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	utilities.LogDebug("Iniciando criação de novo workspace")

	requestingUserUID := r.Context().Value("userUID")
	if requestingUserUID == nil {
		utilities.LogError(fmt.Errorf("UID não encontrado no contexto"), "CreateWorkspaceHandler: Falha na autenticação")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	requestingUserFirebaseUID := requestingUserUID.(string)

	var workspaceInput models.Workspace
	if err := json.NewDecoder(r.Body).Decode(&workspaceInput); err != nil {
		utilities.LogError(err, "CreateWorkspaceHandler: Erro ao decodificar JSON do workspace")
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if workspaceInput.Name == "" {
		utilities.LogError(fmt.Errorf("nome do workspace não fornecido"), "CreateWorkspaceHandler: Validação falhou")
		http.Error(w, "Workspace name is required", http.StatusBadRequest)
		return
	}

	utilities.LogDebug("CreateWorkspaceHandler: Conectando ao banco de dados")
	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "CreateWorkspaceHandler: Erro ao conectar ao banco de dados")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	utilities.LogDebug("CreateWorkspaceHandler: Inserindo novo workspace no banco de dados")
	query := `
		INSERT INTO workspaces (name, description, is_public, owner_uid, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`
	var workspaceID int64
	var createdAt time.Time
	isPublic := true
	err = db.QueryRow(
		query,
		workspaceInput.Name,
		workspaceInput.Description,
		isPublic,
		requestingUserFirebaseUID, // Dono é o usuário que está criando
	).Scan(&workspaceID, &createdAt)

	if err != nil {
		utilities.LogError(err, "CreateWorkspaceHandler: Erro ao criar workspace no banco de dados")
		http.Error(w, "Database error while creating workspace", http.StatusInternalServerError)
		return
	}

	var localUserID int64
	errUser := db.QueryRow("SELECT id FROM users WHERE firebase_uid = $1", requestingUserFirebaseUID).Scan(&localUserID)
	if errUser != nil {
		utilities.LogError(errUser, "CreateWorkspaceHandler: Criador do workspace (UID: "+requestingUserFirebaseUID+") não encontrado no banco de dados users.")
		// Considerar deletar o workspace recém-criado para consistência
		_, delErr := db.Exec("DELETE FROM workspaces WHERE id = $1", workspaceID)
		if delErr != nil {
			utilities.LogError(delErr, "CreateWorkspaceHandler: Falha ao limpar workspace órfão ID: "+strconv.FormatInt(workspaceID, 10))
		}
		http.Error(w, "Erro interno ao associar criador ao workspace", http.StatusInternalServerError)
		return
	}

	utilities.LogDebug("CreateWorkspaceHandler: Adicionando criador como admin do workspace com user_id: %d", localUserID)
	_, err = db.Exec(`
		INSERT INTO workspace_members (workspace_id, user_id, role, joined_at)
		VALUES ($1, $2, 'admin', NOW())
	`, workspaceID, localUserID)

	if err != nil {
		utilities.LogError(err, "CreateWorkspaceHandler: Erro ao adicionar usuário ao workspace_members")
		// Considerar deletar o workspace recém-criado
		_, delErr := db.Exec("DELETE FROM workspaces WHERE id = $1", workspaceID)
		if delErr != nil {
			utilities.LogError(delErr, "CreateWorkspaceHandler: Falha ao limpar workspace órfão ID: "+strconv.FormatInt(workspaceID, 10))
		}
		http.Error(w, "Database error while adding user to workspace", http.StatusInternalServerError)
		return
	}

	// Preparar a resposta com o workspace criado
	createdWorkspace := models.Workspace{
		ID:          workspaceID,
		Name:        workspaceInput.Name,
		Description: workspaceInput.Description,
		IsPublic:    workspaceInput.IsPublic,
		OwnerUID:    requestingUserFirebaseUID,
		CreatedAt:   createdAt,
		Members:     1, // O criador
	}

	utilities.LogInfo("CreateWorkspaceHandler: Workspace criado com sucesso: %s (ID: %d)", createdWorkspace.Name, createdWorkspace.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdWorkspace)
}

// GetWorkspaceInfoHandler busca informações de um workspace específico
func GetWorkspaceInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceIDStr, ok := vars["workspace_id"]
	if !ok {
		utilities.LogError(fmt.Errorf("workspace_id não encontrado nos parâmetros da rota"), "GetWorkspaceInfoHandler: Parâmetro ausente")
		http.Error(w, "Workspace ID is required", http.StatusBadRequest)
		return
	}
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		utilities.LogError(err, "GetWorkspaceInfoHandler: workspace_id inválido")
		http.Error(w, "Invalid Workspace ID format", http.StatusBadRequest)
		return
	}

	requestingUserUID := r.Context().Value("userUID").(string)

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "GetWorkspaceInfoHandler: Erro ao conectar ao banco")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Autorização: Verificar se o usuário é membro do workspace
	isMember, err := models.IsWorkspaceMember(db, requestingUserUID, workspaceID)
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("GetWorkspaceInfoHandler: Erro ao verificar membresia do usuário %s no workspace %d", requestingUserUID, workspaceID))
		http.Error(w, "Failed to verify workspace membership", http.StatusInternalServerError)
		return
	}
	if !isMember {
		// Alternativamente, buscar o workspace e verificar se é público
		wsInfo, errWs := models.GetWorkspaceInfo(db, workspaceID) // Evita erro se !isMember
		if errWs != nil || !wsInfo.IsPublic {
			utilities.InfoLogger.Printf("GetWorkspaceInfoHandler: Usuário %s não autorizado a ver workspace %d (não é membro e não é público)", requestingUserUID, workspaceID)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	workspace, err := models.GetWorkspaceInfo(db, workspaceID)
	if err != nil {
		if err.Error() == "workspace not found" { // Comparando com a string de erro do model
			utilities.LogInfo("GetWorkspaceInfoHandler: Workspace %d não encontrado", workspaceID)
			http.Error(w, "Workspace not found", http.StatusNotFound)
		} else {
			utilities.LogError(err, fmt.Sprintf("GetWorkspaceInfoHandler: Erro ao buscar workspace %d", workspaceID))
			http.Error(w, "Error fetching workspace", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workspace)
}

// UpdateWorkspaceHandler atualiza um workspace existente
func UpdateWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
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

	requestingUserUID := r.Context().Value("userUID").(string)

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		// IsPublic bool `json:"is_public"` // Adicionar se quiser permitir alteração de visibilidade
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if input.Name == "" {
		http.Error(w, "Workspace name cannot be empty", http.StatusBadRequest)
		return
	}

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "UpdateWorkspaceHandler: DB connection error")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Autorização: Somente o dono pode atualizar (ou admin no futuro)
	workspace, err := models.GetWorkspaceInfo(db, workspaceID)
	if err != nil {
		http.Error(w, "Workspace not found or error fetching details", http.StatusNotFound)
		return
	}
	if workspace.OwnerUID != requestingUserUID {
		utilities.InfoLogger.Printf("UpdateWorkspaceHandler: Usuário %s tentou atualizar workspace %d sem ser o dono (%s)", requestingUserUID, workspaceID, workspace.OwnerUID)
		http.Error(w, "Forbidden: Only the owner can update the workspace", http.StatusForbidden)
		return
	}

	err = models.UpdateWorkspace(db, workspaceID, input.Name, input.Description)
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("UpdateWorkspaceHandler: Error updating workspace %d", workspaceID))
		http.Error(w, "Failed to update workspace", http.StatusInternalServerError)
		return
	}

	utilities.LogInfo("UpdateWorkspaceHandler: Workspace %d atualizado com sucesso pelo usuário %s", workspaceID, requestingUserUID)
	w.WriteHeader(http.StatusNoContent) // Ou retornar o workspace atualizado
}

// DeleteWorkspaceHandler deleta um workspace
func DeleteWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
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

	requestingUserUID := r.Context().Value("userUID").(string)

	// 3. Deletar do Firestore
	firestoreClient, err := firebase.GetFirestoreClient() // Sua função para obter o cliente Firestore
	if err != nil {
		utilities.LogError(err, "DeleteWorkspaceHandler: Erro ao obter cliente Firestore")
		http.Error(w, "Failed to connect to Firestore service", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()       // Use o contexto apropriado para sua aplicação
	tasksSubCollectionName := "tasks" // Nome da subcoleção de tarefas no Firestore

	utilities.LogInfo(fmt.Sprintf("DeleteWorkspaceHandler: Iniciando deleção do workspace %d (Firestore ID: %s) do Firestore.", workspaceID, workspaceIDStr))
	err = firebase.DeleteWorkspaceAndSubcollectionsFromFirestore(tasksSubCollectionName, ctx, firestoreClient, workspaceID) // Passa o workspaceID (int64)
	if err != nil {
		// Se a deleção no Firestore falhar, você precisa decidir se continua com a deleção no PG.
		// Por segurança, pode ser melhor parar e logar um erro crítico.
		// Ou, se o erro for "não encontrado", pode ser que já foi deletado, e você pode prosseguir.
		// Para este exemplo, vamos tratar qualquer erro como um problema e não prosseguir com a deleção do PG.
		utilities.LogError(err, fmt.Sprintf("DeleteWorkspaceHandler: Falha ao deletar workspace %d do Firestore. Deleção no PostgreSQL não será realizada.", workspaceID))
		http.Error(w, "Failed to delete workspace data from secondary store. Aborting.", http.StatusInternalServerError)
		return
	}
	utilities.LogInfo(fmt.Sprintf("DeleteWorkspaceHandler: Workspace %d e suas tarefas deletados do Firestore.", workspaceID))

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "DeleteWorkspaceHandler: DB connection error")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// A função models.DeleteWorkspace já verifica se o requestingUserUID é o owner.
	err = models.DeleteWorkspace(db, workspaceID, requestingUserUID)
	if err != nil {
		if err.Error() == "workspace not found or user is not the owner" {
			utilities.InfoLogger.Printf("DeleteWorkspaceHandler: Tentativa de deletar workspace %d por usuário %s (não encontrado ou não é dono)", workspaceID, requestingUserUID)
			http.Error(w, err.Error(), http.StatusForbidden) // Ou StatusNotFound dependendo da mensagem
		} else {
			utilities.LogError(err, fmt.Sprintf("DeleteWorkspaceHandler: Error deleting workspace %d", workspaceID))
			http.Error(w, "Failed to delete workspace", http.StatusInternalServerError)
		}
		return
	}

	utilities.LogInfo("DeleteWorkspaceHandler: Workspace %d deletado com sucesso pelo usuário %s", workspaceID, requestingUserUID)
	w.WriteHeader(http.StatusNoContent)
}

// ListWorkspaceMembersHandler lista membros de um workspace
func ListWorkspaceMembersHandler(w http.ResponseWriter, r *http.Request) {
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

	requestingUserUID := r.Context().Value("userUID").(string)

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "ListWorkspaceMembersHandler: DB connection error")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Autorização: Verificar se o usuário é membro para listar outros membros
	isMember, err := models.IsWorkspaceMember(db, requestingUserUID, workspaceID)
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("ListWorkspaceMembersHandler: Erro ao verificar membresia do usuário %s no workspace %d", requestingUserUID, workspaceID))
		http.Error(w, "Failed to verify workspace membership", http.StatusInternalServerError)
		return
	}
	if !isMember {
		utilities.InfoLogger.Printf("ListWorkspaceMembersHandler: Usuário %s não autorizado a listar membros do workspace %d", requestingUserUID, workspaceID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	members, err := models.ListWorkspaceMembers(db, workspaceID)
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("ListWorkspaceMembersHandler: Erro ao listar membros do workspace %d", workspaceID))
		http.Error(w, "Failed to list workspace members", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// AddUserToWorkspaceHandler adiciona um usuário a um workspace
func AddUserToWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
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

	requestingUserUID := r.Context().Value("userUID").(string)

	var input struct {
		Email string `json:"email"`
		Role  string `json:"role"` // ex: "member", "admin"
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if input.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	// Validação de Role (opcional, pode ser feita no modelo também)
	if input.Role != "admin" && input.Role != "member" {
		input.Role = "member" // Padrão para membro se inválido ou não especificado
	}

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "AddUserToWorkspaceHandler: DB connection error")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Autorização: Somente o dono do workspace (ou um admin) pode adicionar membros.
	// Por simplicidade, vamos checar apenas o dono.
	workspace, err := models.GetWorkspaceInfo(db, workspaceID)
	if err != nil {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	if workspace.OwnerUID != requestingUserUID {
		// Aqui você poderia adicionar uma verificação se o requestingUserUID é um 'admin' do workspace
		// usando uma função como models.GetUserRoleInWorkspace(db, requestingUserUID, workspaceID)
		utilities.InfoLogger.Printf("AddUserToWorkspaceHandler: Usuário %s tentou adicionar membro ao workspace %d sem ser o dono (%s)", requestingUserUID, workspaceID, workspace.OwnerUID)
		http.Error(w, "Forbidden: Only the workspace owner or an admin can add members", http.StatusForbidden)
		return
	}

	err = models.AddUserToWorkspace(db, workspaceID, input.Email, input.Role)
	if err != nil {
		if strings.Contains(err.Error(), "já é membro") || strings.Contains(err.Error(), "usuário com email") || strings.Contains(err.Error(), "não encontrado") {
			utilities.LogInfo(fmt.Sprintf("AddUserToWorkspaceHandler: Falha ao adicionar usuário %s ao workspace %d: %s", input.Email, workspaceID, err.Error()))
			http.Error(w, err.Error(), http.StatusBadRequest) // Erro do cliente se usuário não existe ou já é membro
		} else {
			utilities.LogError(err, fmt.Sprintf("AddUserToWorkspaceHandler: Erro ao adicionar usuário %s ao workspace %d", input.Email, workspaceID))
			http.Error(w, "Failed to add user to workspace", http.StatusInternalServerError)
		}
		return
	}

	utilities.LogInfo("AddUserToWorkspaceHandler: Usuário %s adicionado ao workspace %d com role %s pelo usuário %s", input.Email, workspaceID, input.Role, requestingUserUID)
	w.WriteHeader(http.StatusCreated) // Ou http.StatusOK se preferir
	// Opcional: Retornar o membro adicionado ou uma mensagem de sucesso
	json.NewEncoder(w).Encode(map[string]string{"message": "User added to workspace successfully"})
}

// Em handlers/workspacesHandlers.go

// RemoveUserFromWorkspaceHandler remove um usuário de um workspace,
// esperando o userFirebaseUID no corpo da requisição.
func RemoveUserFromWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceIDStr, okW := vars["workspace_id"]

	if !okW {
		utilities.LogError(fmt.Errorf("workspace_id não encontrado nos parâmetros da rota"), "RemoveUserFromWorkspaceHandler: Parâmetro ausente")
		http.Error(w, "Workspace ID is required in path", http.StatusBadRequest)
		return
	}
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		utilities.LogError(err, "RemoveUserFromWorkspaceHandler: workspace_id inválido na rota")
		http.Error(w, "Invalid Workspace ID format", http.StatusBadRequest)
		return
	}

	requestingUserUID := r.Context().Value("userUID").(string)
	// Pode ser definida no início do seu arquivo handlers/workspacesHandlers.go
	// ou em um arquivo de modelos de requisição, se preferir.
	type RemoveMemberInput struct {
		UserFirebaseUID string `json:"userFirebaseUid"`
	}

	// Decodificar o UserFirebaseUID do corpo da requisição
	var input RemoveMemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utilities.LogError(err, "RemoveUserFromWorkspaceHandler: Erro ao decodificar corpo da requisição")
		http.Error(w, "Invalid request body. Expecting JSON with 'userFirebaseUid'.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	memberFirebaseUID := input.UserFirebaseUID
	if strings.TrimSpace(memberFirebaseUID) == "" {
		utilities.LogError(fmt.Errorf("userFirebaseUid não fornecido no corpo da requisição"), "RemoveUserFromWorkspaceHandler: Dado ausente")
		http.Error(w, "userFirebaseUid is required in request body", http.StatusBadRequest)
		return
	}

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "RemoveUserFromWorkspaceHandler: DB connection error")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Autorização:
	// 1. O usuário a ser removido não pode ser o dono do workspace.
	// 2. Quem remove deve ser o dono do workspace OU o próprio usuário que quer sair.
	workspace, err := models.GetWorkspaceInfo(db, workspaceID)
	if err != nil {
		if err.Error() == "workspace not found" {
			utilities.LogInfo(fmt.Sprintf("RemoveUserFromWorkspaceHandler: Workspace %d não encontrado", workspaceID))
			http.Error(w, "Workspace not found", http.StatusNotFound)
		} else {
			utilities.LogError(err, fmt.Sprintf("RemoveUserFromWorkspaceHandler: Erro ao buscar workspace %d", workspaceID))
			http.Error(w, "Error fetching workspace details", http.StatusInternalServerError)
		}
		return
	}

	if memberFirebaseUID == workspace.OwnerUID {
		// Usando utilities.LogWarn ou similar se você tiver diferentes níveis de log
		utilities.LogInfo(fmt.Sprintf("RemoveUserFromWorkspaceHandler: Usuário %s tentou remover o dono (%s) do workspace %d", requestingUserUID, memberFirebaseUID, workspaceID))
		http.Error(w, "Cannot remove the workspace owner", http.StatusBadRequest)
		return
	}

	isOwner := workspace.OwnerUID == requestingUserUID
	isSelfRemoval := memberFirebaseUID == requestingUserUID

	if !isOwner && !isSelfRemoval {
		// Aqui você poderia adicionar uma verificação se o requestingUserUID é um 'admin' do workspace
		// usando uma função como models.GetUserRoleInWorkspace(db, requestingUserUID, workspaceID)
		utilities.LogInfo(fmt.Sprintf("RemoveUserFromWorkspaceHandler: Usuário %s não autorizado a remover membro %s do workspace %d", requestingUserUID, memberFirebaseUID, workspaceID))
		http.Error(w, "Forbidden: Only workspace owner can remove other members, or user can remove self", http.StatusForbidden)
		return
	}

	err = models.RemoveUserFromWorkspace(db, workspaceID, memberFirebaseUID)
	if err != nil {
		// Tratando erros específicos do model.RemoveUserFromWorkspace
		errMsg := err.Error()
		if strings.Contains(errMsg, "user not found in workspace") ||
			strings.Contains(errMsg, "usuário não encontrado no sistema") ||
			strings.Contains(errMsg, "já removido") {
			utilities.LogInfo(fmt.Sprintf("RemoveUserFromWorkspaceHandler: Falha ao remover usuário %s do workspace %d: %s", memberFirebaseUID, workspaceID, errMsg))
			http.Error(w, errMsg, http.StatusNotFound) // Ou http.StatusBadRequest dependendo do caso
		} else {
			utilities.LogError(err, fmt.Sprintf("RemoveUserFromWorkspaceHandler: Erro ao remover usuário %s do workspace %d", memberFirebaseUID, workspaceID))
			http.Error(w, "Failed to remove user from workspace", http.StatusInternalServerError)
		}
		return
	}

	utilities.LogInfo("RemoveUserFromWorkspaceHandler: Usuário %s removido do workspace %d pelo usuário %s", memberFirebaseUID, workspaceID, requestingUserUID)
	w.WriteHeader(http.StatusNoContent)
}

func ListUserWorkspacesHandler(w http.ResponseWriter, r *http.Request) {
	requestingUserFirebaseUID, ok := r.Context().Value("userUID").(string)
	if !ok || requestingUserFirebaseUID == "" {
		utilities.LogError(fmt.Errorf("userUID não encontrado ou inválido no contexto"), "ListUserWorkspacesHandler: Autenticação falhou")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	utilities.LogInfo(fmt.Sprintf("ListUserWorkspacesHandler: Buscando workspaces para o usuário %s", requestingUserFirebaseUID))

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "ListUserWorkspacesHandler: Erro ao conectar ao banco de dados")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Query para buscar os workspaces do usuário, seu papel, e o owner_uid do workspace
	// A tabela users tem: id (int), firebase_uid (string)
	// A tabela workspaces tem: id (int), name (string), owner_uid (string, references users.firebase_uid)
	// A tabela workspace_members tem: workspace_id (int), user_id (int, references users.id), role (string)
	query := `
		SELECT w.id, w.name, wm.role, w.owner_uid
		FROM workspaces w
		JOIN workspace_members wm ON w.id = wm.workspace_id
		JOIN users u ON wm.user_id = u.id
		WHERE u.firebase_uid = $1
		ORDER BY w.name;
	`
	rows, err := db.Query(query, requestingUserFirebaseUID)
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("ListUserWorkspacesHandler: Erro ao executar query para usuário %s: %v", requestingUserFirebaseUID, err))
		http.Error(w, "Failed to retrieve workspaces", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	userWorkspaces := []models.UserWorkspaceInfo{}
	for rows.Next() {
		var wsi models.UserWorkspaceInfo
		var workspaceOwnerUID string // Para ler o owner_uid do workspace da query
		if err := rows.Scan(&wsi.ID, &wsi.Name, &wsi.UserRole, &workspaceOwnerUID); err != nil {
			utilities.LogError(err, "ListUserWorkspacesHandler: Erro ao escanear linha do workspace")
			// Considerar se deve continuar ou retornar erro aqui. Por ora, continua.
			continue
		}
		wsi.IsOwner = (workspaceOwnerUID == requestingUserFirebaseUID)
		// wsi.OwnerUID = workspaceOwnerUID // Descomente se quiser incluir no JSON de resposta
		userWorkspaces = append(userWorkspaces, wsi)
	}

	if err = rows.Err(); err != nil {
		utilities.LogError(err, "ListUserWorkspacesHandler: Erro após iterar pelas linhas")
		http.Error(w, "Error processing workspace list", http.StatusInternalServerError)
		return
	}

	utilities.LogInfo(fmt.Sprintf("ListUserWorkspacesHandler: Encontrados %d workspaces para o usuário %s", len(userWorkspaces), requestingUserFirebaseUID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userWorkspaces)
}
