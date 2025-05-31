// Em handlers/AiHandler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"projeto-integrador/ai_services" // Onde CallAIAPI, LogAIInteraction, GetContextForIA estão

	// Para GetFirestoreClient em LogAIInteraction
	"projeto-integrador/models"
	"projeto-integrador/utilities"
	"strconv"

	"github.com/gorilla/mux" // Para mux.Vars(r)
)

// Função auxiliar para extrair e validar workspace_id da rota
func getWorkspaceIDFromPath(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	workspaceIDStr, ok := vars["workspace_id"]
	if !ok {
		return 0, fmt.Errorf("workspace_id não encontrado nos parâmetros da rota")
	}
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("formato de workspace_id inválido na rota: %s", workspaceIDStr)
	}
	return workspaceID, nil
}

// WorkspaceTaskAssistantHandler interage com a IA para dar assistência sobre tarefas.
// Rota: /workspace/{workspace_id}/ai/task-assistant
func WorkspaceTaskAssistantHandler(w http.ResponseWriter, r *http.Request) {
	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := r.Context()

	workspaceIDPg, err := getWorkspaceIDFromPath(r)
	if err != nil {
		utilities.LogError(err, "TaskAssistantHandler: Erro ao extrair workspace_id da rota")
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	var frontendInput struct { // WorkspaceID não é mais esperado aqui
		UserMessage string `json:"user_message"`
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

	utilities.LogInfo(fmt.Sprintf("TaskAssistantHandler: Usuário %s pediu assistência para workspace %d com a mensagem: %s",
		requestingUserFirebaseUID, workspaceIDPg, frontendInput.UserMessage))

	workspaceContextForAI, errCtx := ai_services.GetContextForIA(workspaceIDPg, frontendInput.UserMessage)
	if errCtx != nil {
		utilities.LogError(errCtx, "TaskAssistantHandler: Erro ao obter contexto do workspace")
		http.Error(w, `{"error": "Falha ao carregar dados do workspace"}`, http.StatusInternalServerError)
		return
	}

	aiRequestPayload := models.TaskAssistantAIRequest{
		WorkspaceContext: *workspaceContextForAI,
	}
	var aiSuccessfulResponse models.TaskAssistantAIResponse
	aiEndpointPath := "/assistente-tarefas"

	statusCode, rawResponseBodyFromAI, errAI := ai_services.CallAIAPI(ctx, aiEndpointPath, aiRequestPayload, &aiSuccessfulResponse)

	if errAI == nil && statusCode >= 200 && statusCode < 300 {
		// Sucesso na chamada à IA, logar e responder
		ai_services.LogAIInteraction(
			ctx, requestingUserFirebaseUID, workspaceIDPg, "task_assistant",
			frontendInput, aiRequestPayload, aiSuccessfulResponse, statusCode, nil,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(aiSuccessfulResponse)
	} else {
		// Falha na chamada à IA, NÃO logar no histórico de sucessos. Apenas responder ao cliente.
		var errorMsgToClient string
		var errorDetailsForLog string
		if rawResponseBodyFromAI != nil {
			errorDetailsForLog = string(rawResponseBodyFromAI)
			var structuredError models.TaskAssistantAIResponse // Tenta usar a struct de erro
			if json.Unmarshal(rawResponseBodyFromAI, &structuredError) == nil && structuredError.Error != "" {
				errorMsgToClient = fmt.Sprintf(`{"error_ia": "%s"}`, structuredError.Error)
			} else {
				errorMsgToClient = fmt.Sprintf(`{"error": "Erro da API de IA", "details": %q}`, errorDetailsForLog)
			}
		} else if errAI != nil {
			errorDetailsForLog = errAI.Error()
			errorMsgToClient = fmt.Sprintf(`{"error": "Falha na comunicação com o assistente de IA", "details": "%s"}`, errorDetailsForLog)
		} else {
			errorDetailsForLog = "Erro desconhecido da API de IA"
			errorMsgToClient = `{"error": "Erro desconhecido ao contatar o assistente de IA"}`
		}

		utilities.LogError(errAI, fmt.Sprintf("TaskAssistantHandler: Erro da API Python (status: %d) para %s. Resposta: %s", statusCode, aiEndpointPath, errorDetailsForLog))
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errorMsgToClient, statusCode)
	}
}

// CodeReviewAIHandler recebe código do frontend e envia para a API de IA para review.
// Rota: /workspace/{workspace_id}/ai/code-review
func CodeReviewAIHandler(w http.ResponseWriter, r *http.Request) {
	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := r.Context()

	workspaceIDPg, err := getWorkspaceIDFromPath(r)
	if err != nil {
		utilities.LogError(err, "CodeReviewAIHandler: Erro ao extrair workspace_id da rota")
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	var frontendInput models.CodeReviewAIRequest // Não tem mais WorkspaceID aqui
	if err := json.NewDecoder(r.Body).Decode(&frontendInput); err != nil {
		utilities.LogError(err, "CodeReviewAIHandler: Erro ao decodificar JSON de entrada")
		http.Error(w, `{"error": "Corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if frontendInput.Code == "" { /* ... validação ... */
		return
	}
	if frontendInput.Language == "" {
		frontendInput.Language = "Python"
	}

	aiRequestPayload := models.CodeReviewAIRequest{Code: frontendInput.Code, Language: frontendInput.Language}
	var aiSuccessfulResponse models.CodeReviewAIResponse
	aiEndpointPath := "/code-review"

	statusCode, rawResponseBodyFromAI, errAI := ai_services.CallAIAPI(ctx, aiEndpointPath, aiRequestPayload, &aiSuccessfulResponse)

	if errAI == nil && statusCode >= 200 && statusCode < 300 {
		ai_services.LogAIInteraction(
			ctx, requestingUserFirebaseUID, workspaceIDPg, "code_review",
			frontendInput, aiRequestPayload, aiSuccessfulResponse, statusCode, nil,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(aiSuccessfulResponse)
	} else {
		var errorMsgToClient string
		var errorDetailsForLog string
		var structuredError models.CodeReviewAIResponse
		if rawResponseBodyFromAI != nil {
			errorDetailsForLog = string(rawResponseBodyFromAI)
			if json.Unmarshal(rawResponseBodyFromAI, &structuredError) == nil && structuredError.Error != "" {
				errorMsgToClient = fmt.Sprintf(`{"error_ia": "%s"}`, structuredError.Error)
			} else {
				errorMsgToClient = fmt.Sprintf(`{"error": "Erro da API de IA", "details": %q}`, errorDetailsForLog)
			}
		} else if errAI != nil {
			errorDetailsForLog = errAI.Error()
			errorMsgToClient = fmt.Sprintf(`{"error": "Falha na comunicação com a IA", "details": "%s"}`, errorDetailsForLog)
		} else {
			errorDetailsForLog = "Erro desconhecido da API de IA"
			errorMsgToClient = `{"error": "Erro desconhecido ao contatar a IA"}`
		}

		utilities.LogError(errAI, fmt.Sprintf("CodeReviewAIHandler: Erro da API Python (status: %d) para %s. Resposta: %s", statusCode, aiEndpointPath, errorDetailsForLog))
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errorMsgToClient, statusCode)
	}
}

// SummarizeTextAIHandler (adaptação similar)
// Rota: /workspace/{workspace_id}/ai/summarize-text
func SummarizeTextAIHandler(w http.ResponseWriter, r *http.Request) {
	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := r.Context()

	workspaceIDPg, err := getWorkspaceIDFromPath(r)
	if err != nil {
		utilities.LogError(err, "SummarizeTextAIHandler: Erro ao extrair workspace_id da rota")
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	var frontendInput models.SummarizeTextAIRequest // Não tem mais WorkspaceID
	if err := json.NewDecoder(r.Body).Decode(&frontendInput); err != nil {
		utilities.LogError(err, "SummarizeTextAIHandler: Erro ao decodificar JSON")
		http.Error(w, `{"error": "Corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if frontendInput.Text == "" { /* ... validação ... */
		return
	}

	aiRequestPayload := models.SummarizeTextAIRequest{Text: frontendInput.Text}
	var aiSuccessfulResponse models.SummarizeTextAIResponse
	aiEndpointPath := "/summarize-text"

	statusCode, rawResponseBodyFromAI, errAI := ai_services.CallAIAPI(ctx, aiEndpointPath, aiRequestPayload, &aiSuccessfulResponse)

	if errAI == nil && statusCode >= 200 && statusCode < 300 {
		ai_services.LogAIInteraction(
			ctx, requestingUserFirebaseUID, workspaceIDPg, "text_summary",
			frontendInput, aiRequestPayload, aiSuccessfulResponse, statusCode, nil,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(aiSuccessfulResponse)
	} else {
		var errorMsgToClient string
		var errorDetailsForLog string
		var structuredError models.SummarizeTextAIResponse
		if rawResponseBodyFromAI != nil {
			errorDetailsForLog = string(rawResponseBodyFromAI)
			if json.Unmarshal(rawResponseBodyFromAI, &structuredError) == nil && structuredError.Error != "" {
				errorMsgToClient = fmt.Sprintf(`{"error_ia": "%s"}`, structuredError.Error)
			} else {
				errorMsgToClient = fmt.Sprintf(`{"error": "Erro da API de IA", "details": %q}`, errorDetailsForLog)
			}
		} else if errAI != nil {
			errorDetailsForLog = errAI.Error()
			errorMsgToClient = fmt.Sprintf(`{"error": "Falha na comunicação com a IA", "details": "%s"}`, errorDetailsForLog)
		} else {
			errorDetailsForLog = "Erro desconhecido da API de IA"
			errorMsgToClient = `{"error": "Erro desconhecido ao contatar a IA"}`
		}
		utilities.LogError(errAI, fmt.Sprintf("SummarizeTextAIHandler: Erro da API Python (status: %d) para %s. Resposta: %s", statusCode, aiEndpointPath, errorDetailsForLog))
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errorMsgToClient, statusCode)
	}
}

// GenerateMindMapIdeasAIHandler (adaptação similar)
// Rota: /workspace/{workspace_id}/ai/mindmap-ideas
func GenerateMindMapIdeasAIHandler(w http.ResponseWriter, r *http.Request) {
	requestingUserFirebaseUID := r.Context().Value("userUID").(string)
	ctx := r.Context()

	workspaceIDPg, err := getWorkspaceIDFromPath(r)
	if err != nil {
		utilities.LogError(err, "GenerateMindMapIdeasAIHandler: Erro ao extrair workspace_id da rota")
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	var frontendInput models.MindMapIdeasAIRequest // Não tem mais WorkspaceID
	if err := json.NewDecoder(r.Body).Decode(&frontendInput); err != nil {
		utilities.LogError(err, "GenerateMindMapIdeasAIHandler: Erro ao decodificar JSON")
		http.Error(w, `{"error": "Corpo da requisição inválido"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if frontendInput.Text == "" { /* ... validação ... */
		return
	}

	aiRequestPayload := models.MindMapIdeasAIRequest{Text: frontendInput.Text}
	var aiSuccessfulResponse models.MindMapIdeasAIResponse
	aiEndpointPath := "/mindmap-ideas"

	statusCode, rawResponseBodyFromAI, errAI := ai_services.CallAIAPI(ctx, aiEndpointPath, aiRequestPayload, &aiSuccessfulResponse)

	if errAI == nil && statusCode >= 200 && statusCode < 300 {
		ai_services.LogAIInteraction(
			ctx, requestingUserFirebaseUID, workspaceIDPg, "mindmap_ideas",
			frontendInput, aiRequestPayload, aiSuccessfulResponse, statusCode, nil,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(aiSuccessfulResponse)
	} else {
		var errorMsgToClient string
		var errorDetailsForLog string
		var structuredError models.MindMapIdeasAIResponse
		if rawResponseBodyFromAI != nil {
			errorDetailsForLog = string(rawResponseBodyFromAI)
			if json.Unmarshal(rawResponseBodyFromAI, &structuredError) == nil && structuredError.Error != "" {
				errorMsgToClient = fmt.Sprintf(`{"error_ia": "%s"}`, structuredError.Error)
			} else {
				errorMsgToClient = fmt.Sprintf(`{"error": "Erro da API de IA", "details": %q}`, errorDetailsForLog)
			}
		} else if errAI != nil {
			errorDetailsForLog = errAI.Error()
			errorMsgToClient = fmt.Sprintf(`{"error": "Falha na comunicação com a IA", "details": "%s"}`, errorDetailsForLog)
		} else {
			errorDetailsForLog = "Erro desconhecido da API de IA"
			errorMsgToClient = `{"error": "Erro desconhecido ao contatar a IA"}`
		}
		utilities.LogError(errAI, fmt.Sprintf("GenerateMindMapIdeasAIHandler: Erro da API Python (status: %d) para %s. Resposta: %s", statusCode, aiEndpointPath, errorDetailsForLog))
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errorMsgToClient, statusCode)
	}
}
