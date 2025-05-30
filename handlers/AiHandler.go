// Em handlers/AiHandler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"projeto-integrador/ai_services" // Onde CallAIAPI e LogAIInteraction, GetContextForIA estão

	// Para GetFirestoreClient (se LogAIInteraction precisar diretamente)
	"projeto-integrador/models" // Onde as structs de request/response da IA e AIRequestHistoryEntry estão
	"projeto-integrador/utilities"
	"strconv"
	// "github.com/gorilla/mux" // Desnecessário se não estiver usando mux.Vars diretamente aqui
)

// WorkspaceTaskAssistantHandler interage com a IA para dar assistência sobre tarefas.
func WorkspaceTaskAssistantHandler(w http.ResponseWriter, r *http.Request) {
	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := r.Context()

	var frontendInput struct {
		UserMessage string `json:"user_message"`
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&frontendInput); err != nil {
		utilities.LogError(err, "TaskAssistantHandler: Erro ao decodificar JSON de entrada")
		http.Error(w, `{"error": "Corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if frontendInput.UserMessage == "" {
		http.Error(w, `{"error": "Mensagem do usuário é obrigatória"}`, http.StatusBadRequest)
		return
	}
	if frontendInput.WorkspaceID == "" {
		http.Error(w, `{"error": "ID do workspace é obrigatório"}`, http.StatusBadRequest)
		return
	}

	workspaceIDPg, convErr := strconv.ParseInt(frontendInput.WorkspaceID, 10, 64)
	if convErr != nil {
		utilities.LogError(convErr, "TaskAssistantHandler: WorkspaceID inválido no payload: "+frontendInput.WorkspaceID)
		http.Error(w, `{"error": "Formato de WorkspaceID inválido"}`, http.StatusBadRequest)
		return
	}

	utilities.LogInfo(fmt.Sprintf("TaskAssistantHandler: Usuário %s pediu assistência para workspace %s com a mensagem: %s",
		requestingUserFirebaseUID, frontendInput.WorkspaceID, frontendInput.UserMessage))

	// 1. Montar o contexto para a IA
	workspaceContextForAI, errCtx := ai_services.GetContextForIA(workspaceIDPg, frontendInput.UserMessage)
	if errCtx != nil {
		utilities.LogError(errCtx, "TaskAssistantHandler: Erro ao obter contexto do workspace")
		http.Error(w, `{"error": "Falha ao carregar dados do workspace"}`, http.StatusInternalServerError)
		return
	}

	// 2. Preparar o payload para a API Python
	aiRequestPayload := models.TaskAssistantAIRequest{
		WorkspaceContext: *workspaceContextForAI,
	}

	var aiSuccessfulResponse models.TaskAssistantAIResponse
	aiEndpointPath := "/assistente-tarefas" // CONFIRME ESTE PATH COM SUA API PYTHON

	// 3. Chamar a API de IA
	statusCode, rawResponseBodyFromAI, errAI := ai_services.CallAIAPI(ctx, aiEndpointPath, aiRequestPayload, &aiSuccessfulResponse)

	// 4. Tentar decodificar a resposta bruta para log (seja sucesso ou erro estruturado da IA)
	var responseToLog interface{}
	if errAI == nil && statusCode >= 200 && statusCode < 300 {
		responseToLog = aiSuccessfulResponse
	} else if rawResponseBodyFromAI != nil {
		var genericErrorResponse map[string]interface{}
		if json.Unmarshal(rawResponseBodyFromAI, &genericErrorResponse) == nil {
			responseToLog = genericErrorResponse
		} else {
			responseToLog = string(rawResponseBodyFromAI)
		}
	}

	// 5. Logar a interação
	ai_services.LogAIInteraction( // Removido firebase.GetFirestoreClient() daqui, pois LogAIInteraction já o obtém
		ctx,
		requestingUserFirebaseUID,
		workspaceIDPg,
		"task_assistant",
		frontendInput,
		aiRequestPayload,
		responseToLog,
		statusCode,
		errAI,
	)

	// 6. Responder ao frontend
	if errAI != nil {
		utilities.LogError(errAI, fmt.Sprintf("TaskAssistantHandler: Erro da API Python (status: %d) para /assistente-tarefas. Corpo: %s", statusCode, string(rawResponseBodyFromAI)))
		w.Header().Set("Content-Type", "application/json")
		// Tenta retornar o erro da IA de forma estruturada se possível, senão o erro da chamada
		if responseToLog != nil {
			// Se responseToLog for um mapa (erro estruturado da IA), envie isso.
			if errMap, ok := responseToLog.(map[string]interface{}); ok {
				if errMsg, ok := errMap["error"].(string); ok {
					http.Error(w, fmt.Sprintf(`{"error_ia": "%s"}`, errMsg), statusCode)
					return
				}
			}
		}
		http.Error(w, fmt.Sprintf(`{"error": "Falha na comunicação com o assistente de IA", "details": %q}`, string(rawResponseBodyFromAI)), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(aiSuccessfulResponse)
}

// CodeReviewAIHandler, SummarizeTextAIHandler, GenerateMindMapIdeasAIHandler
// Precisam ser ajustados de forma similar para incluir o logging com LogAIInteraction
// e para o tratamento de erro ao decodificar a resposta da IA.

// Exemplo ajustado para CodeReviewAIHandler:
func CodeReviewAIHandler(w http.ResponseWriter, r *http.Request) {
	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := r.Context()
	var workspaceIDForLog int64 = 0 // Ou obtenha de algum lugar se for relevante

	var frontendInput models.CodeReviewAIRequest
	if err := json.NewDecoder(r.Body).Decode(&frontendInput); err != nil {
		utilities.LogError(err, "CodeReviewAIHandler: Erro ao decodificar JSON de entrada")
		http.Error(w, `{"error": "Corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if frontendInput.Code == "" {
		http.Error(w, `{"error": "O campo 'code' é obrigatório para code review"}`, http.StatusBadRequest)
		return
	}
	if frontendInput.Language == "" {
		frontendInput.Language = "Python" // Default ou tornar obrigatório
	}

	aiRequestPayload := models.CodeReviewAIRequest{Code: frontendInput.Code, Language: frontendInput.Language}
	var aiSuccessfulResponse models.CodeReviewAIResponse
	aiEndpointPath := "/code-review" // CONFIRME ESTE PATH

	statusCode, rawResponseBodyFromAI, errAI := ai_services.CallAIAPI(ctx, aiEndpointPath, aiRequestPayload, &aiSuccessfulResponse)

	var responseToLog interface{}
	var errorDetailFromAI models.CodeReviewAIResponse // Para tentar pegar o .Error da IA
	if errAI == nil && statusCode >= 200 && statusCode < 300 {
		responseToLog = aiSuccessfulResponse
	} else if rawResponseBodyFromAI != nil {
		if json.Unmarshal(rawResponseBodyFromAI, &errorDetailFromAI) == nil && errorDetailFromAI.Error != "" {
			responseToLog = errorDetailFromAI
		} else {
			var genericErrorResponse map[string]interface{}
			if json.Unmarshal(rawResponseBodyFromAI, &genericErrorResponse) == nil {
				responseToLog = genericErrorResponse
			} else {
				responseToLog = string(rawResponseBodyFromAI)
			}
		}
	}

	ai_services.LogAIInteraction(
		ctx,
		requestingUserFirebaseUID,
		workspaceIDForLog, // Usar ID real se aplicável
		"code_review",
		frontendInput,
		aiRequestPayload,
		responseToLog,
		statusCode,
		errAI,
	)

	if errAI != nil {
		utilities.LogError(errAI, fmt.Sprintf("CodeReviewAIHandler: Erro da API Python (status: %d) para %s. Corpo: %s", statusCode, aiEndpointPath, string(rawResponseBodyFromAI)))
		w.Header().Set("Content-Type", "application/json")
		if errorDetailFromAI.Error != "" {
			http.Error(w, fmt.Sprintf(`{"error_ia": "%s"}`, errorDetailFromAI.Error), statusCode)
		} else {
			http.Error(w, fmt.Sprintf(`{"error": "Falha ao processar revisão de código", "details": %q}`, string(rawResponseBodyFromAI)), statusCode)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(aiSuccessfulResponse)
}

// Adapte SummarizeTextAIHandler e GenerateMindMapIdeasAIHandler de forma similar:
// 1. Adicione a obtenção do requestingUserFirebaseUID e ctx.
// 2. Defina um workspaceIDForLog apropriado.
// 3. Após chamar CallAIAPI, determine responseToLog (sucesso ou erro estruturado/bruto).
// 4. Chame aiservices.LogAIInteraction.
// 5. Ajuste a resposta de erro ao frontend para incluir detalhes do erro da IA, se possível.

func SummarizeTextAIHandler(w http.ResponseWriter, r *http.Request) {
	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := r.Context()
	var workspaceIDForLog int64 = 0 // Ou ID real se aplicável

	var frontendInput models.SummarizeTextAIRequest
	if err := json.NewDecoder(r.Body).Decode(&frontendInput); err != nil {
		utilities.LogError(err, "SummarizeTextAIHandler: Erro ao decodificar JSON")
		http.Error(w, `{"error": "Corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if frontendInput.Text == "" {
		http.Error(w, `{"error": "O campo 'text' é obrigatório para resumo"}`, http.StatusBadRequest)
		return
	}

	aiRequestPayload := models.SummarizeTextAIRequest{Text: frontendInput.Text}
	var aiSuccessfulResponse models.SummarizeTextAIResponse
	var errorDetailFromAI models.SummarizeTextAIResponse
	aiEndpointPath := "/summarize" // CONFIRME ESTE PATH

	statusCode, rawResponseBodyFromAI, errAI := ai_services.CallAIAPI(ctx, aiEndpointPath, aiRequestPayload, &aiSuccessfulResponse)

	var responseToLog interface{}
	if errAI == nil && statusCode >= 200 && statusCode < 300 {
		responseToLog = aiSuccessfulResponse
	} else if rawResponseBodyFromAI != nil {
		if json.Unmarshal(rawResponseBodyFromAI, &errorDetailFromAI) == nil && errorDetailFromAI.Error != "" {
			responseToLog = errorDetailFromAI
		} else {
			var genericErrorResponse map[string]interface{}
			if json.Unmarshal(rawResponseBodyFromAI, &genericErrorResponse) == nil {
				responseToLog = genericErrorResponse
			} else {
				responseToLog = string(rawResponseBodyFromAI)
			}
		}
	}

	ai_services.LogAIInteraction(
		ctx,
		requestingUserFirebaseUID,
		workspaceIDForLog,
		"text_summary",
		frontendInput,
		aiRequestPayload,
		responseToLog,
		statusCode,
		errAI,
	)

	if errAI != nil {
		utilities.LogError(errAI, fmt.Sprintf("SummarizeTextAIHandler: Erro da API Python (status: %d) para %s. Corpo: %s", statusCode, aiEndpointPath, string(rawResponseBodyFromAI)))
		w.Header().Set("Content-Type", "application/json")
		if errorDetailFromAI.Error != "" {
			http.Error(w, fmt.Sprintf(`{"error_ia": "%s"}`, errorDetailFromAI.Error), statusCode)
		} else {
			http.Error(w, fmt.Sprintf(`{"error": "Falha ao processar resumo", "details": %q}`, string(rawResponseBodyFromAI)), statusCode)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(aiSuccessfulResponse)
}

func GenerateMindMapIdeasAIHandler(w http.ResponseWriter, r *http.Request) {
	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := r.Context()
	var workspaceIDForLog int64 = 0 // Ou ID real se aplicável

	var frontendInput models.MindMapIdeasAIRequest
	if err := json.NewDecoder(r.Body).Decode(&frontendInput); err != nil {
		utilities.LogError(err, "GenerateMindMapIdeasAIHandler: Erro ao decodificar JSON")
		http.Error(w, `{"error": "Corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if frontendInput.Text == "" {
		http.Error(w, `{"error": "O campo 'text' é obrigatório para mapa mental"}`, http.StatusBadRequest)
		return
	}

	aiRequestPayload := models.MindMapIdeasAIRequest{Text: frontendInput.Text}
	var aiSuccessfulResponse models.MindMapIdeasAIResponse
	var errorDetailFromAI models.MindMapIdeasAIResponse
	aiEndpointPath := "/mindmap-ideas" // CONFIRME ESTE PATH

	statusCode, rawResponseBodyFromAI, errAI := ai_services.CallAIAPI(ctx, aiEndpointPath, aiRequestPayload, &aiSuccessfulResponse)

	var responseToLog interface{}
	if errAI == nil && statusCode >= 200 && statusCode < 300 {
		responseToLog = aiSuccessfulResponse
	} else if rawResponseBodyFromAI != nil {
		if json.Unmarshal(rawResponseBodyFromAI, &errorDetailFromAI) == nil && errorDetailFromAI.Error != "" {
			responseToLog = errorDetailFromAI
		} else {
			var genericErrorResponse map[string]interface{}
			if json.Unmarshal(rawResponseBodyFromAI, &genericErrorResponse) == nil {
				responseToLog = genericErrorResponse
			} else {
				responseToLog = string(rawResponseBodyFromAI)
			}
		}
	}

	ai_services.LogAIInteraction(
		ctx,
		requestingUserFirebaseUID,
		workspaceIDForLog,
		"mindmap_ideas",
		frontendInput,
		aiRequestPayload,
		responseToLog,
		statusCode,
		errAI,
	)

	if errAI != nil {
		utilities.LogError(errAI, fmt.Sprintf("GenerateMindMapIdeasAIHandler: Erro da API Python (status: %d) para %s. Corpo: %s", statusCode, aiEndpointPath, string(rawResponseBodyFromAI)))
		w.Header().Set("Content-Type", "application/json")
		if errorDetailFromAI.Error != "" {
			http.Error(w, fmt.Sprintf(`{"error_ia": "%s"}`, errorDetailFromAI.Error), statusCode)
		} else {
			http.Error(w, fmt.Sprintf(`{"error": "Falha ao gerar ideias para mapa mental", "details": %q}`, string(rawResponseBodyFromAI)), statusCode)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(aiSuccessfulResponse)
}
