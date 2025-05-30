package ai_services

import (
	"context"
	"fmt"
	"projeto-integrador/firebase" // Onde GetFirestoreClient está
	"projeto-integrador/models"   // Onde AIRequestHistoryEntry está
	"projeto-integrador/utilities"
	"strconv"
	"time" // Para o campo Timestamp na struct
	// Para firestore.ServerTimestamp
)

// LogAIInteraction registra uma interação com a API de IA no Firestore.
func LogAIInteraction(
	ctx context.Context,
	userID string, // Firebase UID do usuário requisitante
	workspaceIDPg int64, // ID do workspace no PostgreSQL
	serviceType string, // Tipo de serviço de IA (ex: "code_review")
	frontendPayload interface{}, // O que o frontend enviou para o backend Go
	requestToAI interface{}, // O que o backend Go enviou para a API Python
	responseFromAI interface{}, // A resposta da API Python (pode ser a struct de sucesso ou de erro)
	aiStatusCode int, // Status code da resposta da API Python
	aiCallError error, // Erro ocorrido na chamada à API Python (se houver)
) {
	firestoreClient, err := firebase.GetFirestoreClient()
	if err != nil {
		utilities.LogError(err, "LogAIInteraction: Falha ao obter cliente Firestore")
		return // Não impede o fluxo principal, apenas não loga
	}

	workspaceDocIDForFirestore := strconv.FormatInt(workspaceIDPg, 10)
	historyCollectionPath := fmt.Sprintf("workspaces/%s/ai_request_history", workspaceDocIDForFirestore)

	entry := models.AIRequestHistoryEntry{
		UserID:        userID,
		WorkspaceIDPg: workspaceIDPg,
		AIServiceType: serviceType,
		Timestamp:     time.Now(), // O SDK Go converte para Timestamp do Firestore.
		// Para usar o timestamp do servidor:
		// Troque o tipo de Timestamp na struct para interface{}
		// e aqui atribua: Timestamp: firestore.ServerTimestamp,
		FrontendRequestPayload: frontendPayload,
		RequestToAI:            requestToAI,
		ResponseFromAI:         responseFromAI, // Se errAI não for nil, isso pode conter a estrutura de erro da IA
		AIStatusCode:           aiStatusCode,
	}

	if aiCallError != nil {
		entry.AIError = aiCallError.Error()
	}

	docRef, _, err := firestoreClient.Collection(historyCollectionPath).Add(ctx, entry)
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("LogAIInteraction: Falha ao salvar histórico de IA para workspace %s, user %s", workspaceDocIDForFirestore, userID))
	} else {
		utilities.LogDebug(fmt.Sprintf("LogAIInteraction: Histórico de IA salvo com ID %s para workspace %s", docRef.ID, workspaceDocIDForFirestore))
	}
}
