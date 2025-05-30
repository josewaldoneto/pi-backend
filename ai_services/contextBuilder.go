package ai_services

import (
	"context"
	"fmt"
	"projeto-integrador/database"
	"projeto-integrador/firebase" // Para GetFirestoreClient
	"projeto-integrador/models"
	"projeto-integrador/utilities"
	"strconv"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const maxTasksForAIContext = 15 // Limite de tarefas para enviar no contexto da IA

// listTasksForAIContext busca tarefas do Firestore para um workspace específico,
// formatando-as para o contexto da IA.
func listTasksForAIContext(ctx context.Context, firestoreClient *firestore.Client, workspaceIDPg int64, limit int) ([]models.TarefaContext, error) {
	workspaceDocIDForFirestore := strconv.FormatInt(workspaceIDPg, 10)
	tasksSubCollectionName := "tasks" // Conforme definido em tasksHandlers

	// utilities.LogDebug(fmt.Sprintf("listTasksForAIContext: Buscando até %d tarefas do workspace %s no Firestore", limit, workspaceDocIDForFirestore))

	iter := firestoreClient.Collection("workspaces").Doc(workspaceDocIDForFirestore).Collection(tasksSubCollectionName).
		OrderBy("lastUpdatedAt", firestore.Desc). // Ordenar por mais recentes (ou outro critério)
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	var tarefasCtx []models.TarefaContext
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			utilities.LogError(err, fmt.Sprintf("listTasksForAIContext: Erro ao iterar tarefas do Firestore para workspace %s", workspaceDocIDForFirestore))
			return nil, fmt.Errorf("erro ao buscar tarefas do Firestore: %w", err)
		}

		var taskDetail models.TaskDetailsFirestore
		if err := doc.DataTo(&taskDetail); err != nil {
			utilities.LogInfo(fmt.Sprintf("listTasksForAIContext: Erro ao converter dados da tarefa Firestore para struct (ID: %s): %v", doc.Ref.ID, err))
			continue // Pula esta tarefa se houver erro de conversão
		}

		tarefasCtx = append(tarefasCtx, models.TarefaContext{
			Titulo:     taskDetail.Title,
			Status:     taskDetail.Status,
			Prioridade: taskDetail.Priority,
		})
	}
	// utilities.LogDebug(fmt.Sprintf("listTasksForAIContext: %d tarefas formatadas para o contexto da IA.", len(tarefasCtx)))
	return tarefasCtx, nil
}

// GetContextForIA busca e formata os dados de um workspace para a IA.
// workspaceIDPg é o ID numérico do workspace no PostgreSQL.
// userMessage é a mensagem/prompt atual do usuário.
func GetContextForIA(workspaceIDPg int64, userMessage string) (*models.IAWorkspaceContext, error) {
	ctx := context.Background() // Use um contexto apropriado

	// utilities.LogDebug(fmt.Sprintf("GetContextForIA: Montando contexto para workspace ID PG: %d", workspaceIDPg))

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("GetContextForIA: Erro ao conectar ao PG para workspace %d", workspaceIDPg))
		return nil, err
	}
	defer db.Close()

	// 1. Buscar informações do Workspace do PostgreSQL
	wsInfo, err := models.GetWorkspaceInfo(db, workspaceIDPg) // Esta função já busca o nome, descrição, etc.
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("GetContextForIA: Erro ao buscar info do workspace %d do PG", workspaceIDPg))
		return nil, err
	}

	// 2. Buscar membros do Workspace do PostgreSQL
	wsMembersModels, err := models.ListWorkspaceMembers(db, workspaceIDPg) // Esta função retorna []models.WorkspaceMember
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("GetContextForIA: Erro ao buscar membros do workspace %d do PG", workspaceIDPg))
		return nil, err
	}
	usuariosCtx := make([]models.UsuarioContext, len(wsMembersModels))
	for i, member := range wsMembersModels {
		usuariosCtx[i] = models.UsuarioContext{
			Nome: member.DisplayName, // models.WorkspaceMember tem DisplayName
			Role: member.Role,
		}
	}

	// 3. Buscar tarefas recentes/relevantes do Firestore
	firestoreClient, err := firebase.GetFirestoreClient()
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("GetContextForIA: Erro ao obter cliente Firestore para workspace %d", workspaceIDPg))
		return nil, err
	}
	tarefasCtx, err := listTasksForAIContext(ctx, firestoreClient, workspaceIDPg, maxTasksForAIContext)
	if err != nil {
		// Não tratar como erro fatal, a IA pode funcionar com contexto parcial (sem tarefas)
		utilities.LogInfo(fmt.Sprintf("GetContextForIA: Não foi possível buscar tarefas para o contexto da IA para o workspace %d: %v", workspaceIDPg, err))
		tarefasCtx = []models.TarefaContext{} // Envia lista vazia
	}

	contexto := &models.IAWorkspaceContext{
		WorkspaceIDStr: strconv.FormatInt(workspaceIDPg, 10),
		GrupoNome:      wsInfo.Name,
		DescricaoGrupo: wsInfo.Description,
		Usuarios:       usuariosCtx,
		Tarefas:        tarefasCtx,
		MsgDoUsuario:   userMessage,
	}

	// utilities.LogDebug(fmt.Sprintf("GetContextForIA: Contexto montado para workspace %d: %+v", workspaceIDPg, contexto))
	return contexto, nil
}
