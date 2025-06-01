package ai_services

import (
	"context"
	"fmt"
	"projeto-integrador/database"  // Para conectar ao PostgreSQL
	"projeto-integrador/firebase"  // Onde GetFirestoreClient() está
	"projeto-integrador/models"    // Onde todas as suas structs de modelo estão
	"projeto-integrador/utilities" // Seu pacote de logging
	"strconv"                      // Para converter int64 para string

	"cloud.google.com/go/firestore"  // Para firestore.Desc e outras funcionalidades do Firestore
	"google.golang.org/api/iterator" // Para iterator.Done
)

const maxTasksForAIContext = 15        // Limite de tarefas para enviar no contexto da IA
const tasksSubCollectionName = "tasks" // Nome da subcoleção de tarefas no Firestore

// listTasksForAIContext busca tarefas do Firestore para um workspace específico,
// formatando-as para o contexto da IA.
// workspaceIDPg é o ID NUMÉRICO do workspace no PostgreSQL.
func listTasksForAIContext(ctx context.Context, firestoreClient *firestore.Client, workspaceIDPg int64, limit int) ([]models.TarefaContext, error) {
	workspaceDocIDForFirestore := strconv.FormatInt(workspaceIDPg, 10)
	fullPathToTasks := fmt.Sprintf("workspaces/%s/%s", workspaceDocIDForFirestore, tasksSubCollectionName)

	utilities.LogDebug(fmt.Sprintf("listTasksForAIContext: Iniciando busca. Path: %s, OrderBy: 'last_updated_at' Desc, Limit: %d", fullPathToTasks, limit))

	iter := firestoreClient.Collection("workspaces").Doc(workspaceDocIDForFirestore).Collection(tasksSubCollectionName).
		OrderBy("last_updated_at", firestore.Desc). // Certifique-se que este é o nome do campo no Firestore
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	var tarefasCtx []models.TarefaContext
	docCount := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			utilities.LogDebug(fmt.Sprintf("listTasksForAIContext: Fim da iteração. Documentos processados: %d", docCount))
			break
		}
		if err != nil {
			// Este é um erro durante a iteração, ANTES de chegar ao fim.
			utilities.LogError(err, fmt.Sprintf("listTasksForAIContext: Erro REAL ao iterar tarefas do Firestore para path %s", fullPathToTasks))
			return nil, fmt.Errorf("erro ao buscar tarefas do Firestore: %w", err)
		}

		docCount++
		utilities.LogDebug(fmt.Sprintf("listTasksForAIContext: Documento encontrado - ID: %s, Path: %s", doc.Ref.ID, doc.Ref.Path))

		var taskDetail models.TaskDetailsFirestore
		if errDataTo := doc.DataTo(&taskDetail); errDataTo != nil {
			// Log detalhado do erro de conversão e dos dados brutos do documento se possível
			rawData := doc.Data()
			utilities.LogError(errDataTo, fmt.Sprintf("listTasksForAIContext: Erro ao converter dados da tarefa Firestore para struct (Doc ID: %s). Dados Brutos: %+v", doc.Ref.ID, rawData))
			continue // Pula esta tarefa, mas loga o problema
		}

		utilities.LogDebug(fmt.Sprintf("listTasksForAIContext: Tarefa convertida com sucesso - Título: %s, Status: %s", taskDetail.Title, taskDetail.Status))
		tarefasCtx = append(tarefasCtx, models.TarefaContext{
			Titulo:     taskDetail.Title,
			Status:     taskDetail.Status,
			Descricao:  taskDetail.Description, // Se necessário, pode ser uma descrição curta
			Prioridade: taskDetail.Priority,
		})
	}

	if len(tarefasCtx) == 0 && docCount > 0 {
		utilities.LogInfo(fmt.Sprintf("listTasksForAIContext: %d documentos foram iterados, mas a lista de TarefaContext está vazia. Verifique erros de DataTo.", docCount))
	}
	utilities.LogDebug(fmt.Sprintf("listTasksForAIContext: Finalizado. %d tarefas formatadas para o contexto da IA para o workspace ID PG %d.", len(tarefasCtx), workspaceIDPg))
	return tarefasCtx, nil
}

// GetContextForIA busca e formata os dados de um workspace para a IA.
// workspaceIDPg é o ID numérico do workspace no PostgreSQL.
// userMessage é a mensagem/prompt atual do usuário.
func GetContextForIA(workspaceIDPg int64, userMessage string) (*models.IAWorkspaceContext, error) {
	ctx := context.Background() // Use um contexto apropriado para suas chamadas

	utilities.LogDebug(fmt.Sprintf("GetContextForIA: Montando contexto para workspace ID PG: %d, Mensagem: '%s'", workspaceIDPg, userMessage))

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("GetContextForIA: Erro ao conectar ao PG para workspace %d", workspaceIDPg))
		return nil, err
	}
	defer db.Close()

	// 1. Buscar informações do Workspace do PostgreSQL
	wsInfo, err := models.GetWorkspaceInfo(db, workspaceIDPg) // Retorna *models.Workspace
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("GetContextForIA: Erro ao buscar info do workspace %d do PG", workspaceIDPg))
		return nil, err
	}
	utilities.LogDebug(fmt.Sprintf("GetContextForIA: Informações do workspace '%s' obtidas do PG.", wsInfo.Name))

	// 2. Buscar membros do Workspace do PostgreSQL
	wsMembersModels, err := models.ListWorkspaceMembers(db, workspaceIDPg) // Retorna []models.WorkspaceMember
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("GetContextForIA: Erro ao buscar membros do workspace %d do PG", workspaceIDPg))
		return nil, err
	}
	usuariosCtx := make([]models.UsuarioContext, len(wsMembersModels))
	for i, member := range wsMembersModels {
		usuariosCtx[i] = models.UsuarioContext{
			Nome: member.DisplayName,
			Role: member.Role,
		}
	}
	utilities.LogDebug(fmt.Sprintf("GetContextForIA: %d membros do workspace formatados para o contexto.", len(usuariosCtx)))

	// 3. Buscar tarefas recentes/relevantes do Firestore
	firestoreClient, err := firebase.GetFirestoreClient() // Assume que esta função está no pacote firebase
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("GetContextForIA: Erro ao obter cliente Firestore para workspace %d", workspaceIDPg))
		return nil, err // Se não conseguir o cliente Firestore, não podemos buscar tarefas
	}

	tarefasCtx, err := listTasksForAIContext(ctx, firestoreClient, workspaceIDPg, maxTasksForAIContext)
	if err != nil {
		// Decidimos anteriormente não tratar isso como um erro fatal para o GetContextForIA,
		// mas vamos logar o erro que veio de listTasksForAIContext.
		utilities.LogInfo(fmt.Sprintf("GetContextForIA: Não foi possível buscar tarefas do Firestore para o contexto da IA para o workspace %d: %v. Continuando com lista de tarefas vazia.", workspaceIDPg, err))
		tarefasCtx = []models.TarefaContext{} // Envia lista vazia se houve erro
	}
	utilities.LogDebug(fmt.Sprintf("GetContextForIA: %d tarefas obtidas do Firestore para o contexto.", len(tarefasCtx)))

	contexto := &models.IAWorkspaceContext{
		WorkspaceIDStr: strconv.FormatInt(workspaceIDPg, 10), // ID do workspace do PG como string
		GrupoNome:      wsInfo.Name,
		DescricaoGrupo: wsInfo.Description,
		Usuarios:       usuariosCtx,
		Tarefas:        tarefasCtx, // Aqui entram as tarefas buscadas do Firestore
		MsgDoUsuario:   userMessage,
	}

	utilities.LogDebug(fmt.Sprintf("GetContextForIA: Contexto final montado para workspace %d.", workspaceIDPg))
	return contexto, nil
}
