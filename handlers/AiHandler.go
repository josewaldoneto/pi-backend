// Em handlers/aiSpecificHandlers.go (novo arquivo)
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	ai_service "projeto-integrador/ai_services" // Pacote onde você definiu CallAIAPI
	"projeto-integrador/models"                 // Pacote onde você definiu as structs de request/response da IA
	"projeto-integrador/utilities"
)

// CodeReviewAIHandler recebe código do frontend e envia para a API de IA para review.
func CodeReviewAIHandler(w http.ResponseWriter, r *http.Request) {
	var input models.CodeReviewAIRequest // Struct que seu frontend envia
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utilities.LogError(err, "CodeReviewAIHandler: Erro ao decodificar JSON de entrada")
		http.Error(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if input.Code == "" {
		http.Error(w, `{"error": "O campo 'code' é obrigatório para code review"}`, http.StatusBadRequest)
		return
	}
	if input.Language == "" { // Pode adicionar validação para linguagens suportadas
		input.Language = "Go" // Um padrão, ou tornar obrigatório
	}

	var aiResponse models.CodeReviewAIResponse
	var aiErrorResponse models.CodeReviewAIResponse // Usando a mesma struct se o erro da IA vier no campo 'error'

	// O payload para a API de IA pode ser o mesmo que o input do frontend, ou você pode mapeá-lo.
	// Aqui, estamos assumindo que a API de IA espera a mesma estrutura.
	aiRequestPayload := models.CodeReviewAIRequest{Code: input.Code, Language: input.Language}

	statusCode, err := ai_service.CallAIAPI(r.Context(), "/code-review", aiRequestPayload, &aiResponse, &aiErrorResponse)
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("CodeReviewAIHandler: Erro ao chamar API de IA (status: %d)", statusCode))
		// Se aiErrorResponse.Error tiver algo, use-o
		if aiErrorResponse.Error != "" {
			http.Error(w, fmt.Sprintf(`{"error": "Erro da IA: %s"}`, aiErrorResponse.Error), statusCode)
		} else {
			http.Error(w, `{"error": "Falha ao processar revisão de código"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode) // Geralmente http.StatusOK se chegou aqui sem erro
	json.NewEncoder(w).Encode(aiResponse)
}

// SummarizeTextAIHandler recebe texto e envia para a API de IA para resumo.
func SummarizeTextAIHandler(w http.ResponseWriter, r *http.Request) {
	var input models.SummarizeTextAIRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utilities.LogError(err, "SummarizeTextAIHandler: Erro ao decodificar JSON")
		http.Error(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if input.Text == "" {
		http.Error(w, `{"error": "O campo 'text' é obrigatório para resumo"}`, http.StatusBadRequest)
		return
	}

	var aiResponse models.SummarizeTextAIResponse
	var aiErrorResponse models.SummarizeTextAIResponse // Reutilizando se o campo 'error' é comum

	aiRequestPayload := models.SummarizeTextAIRequest{Text: input.Text}

	statusCode, err := ai_service.CallAIAPI(r.Context(), "/summarize", aiRequestPayload, &aiResponse, &aiErrorResponse)
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("SummarizeTextAIHandler: Erro ao chamar API de IA (status: %d)", statusCode))
		if aiErrorResponse.Error != "" {
			http.Error(w, fmt.Sprintf(`{"error": "Erro da IA: %s"}`, aiErrorResponse.Error), statusCode)
		} else {
			http.Error(w, `{"error": "Falha ao processar resumo de texto"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(aiResponse)
}

// GenerateMindMapIdeasAIHandler recebe texto e envia para a API de IA para ideias de mapa mental.
func GenerateMindMapIdeasAIHandler(w http.ResponseWriter, r *http.Request) {
	// Similar aos handlers acima, mas usando MindMapIdeasAIRequest e MindMapIdeasAIResponse
	var input models.MindMapIdeasAIRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		// ...
		http.Error(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if input.Text == "" {
		http.Error(w, `{"error": "O campo 'text' é obrigatório para mapa mental"}`, http.StatusBadRequest)
		return
	}

	var aiResponse models.MindMapIdeasAIResponse
	var aiErrorResponse models.MindMapIdeasAIResponse

	aiRequestPayload := models.MindMapIdeasAIRequest{Text: input.Text}

	statusCode, err := ai_service.CallAIAPI(r.Context(), "/mindmap-ideas", aiRequestPayload, &aiResponse, &aiErrorResponse)
	if err != nil {
		// ... (tratamento de erro similar) ...
		http.Error(w, `{"error": "Falha ao gerar ideias para mapa mental"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(aiResponse)
}

// WorkspaceTaskAssistantHandler recebe uma mensagem do usuário e o contexto do workspace,
// envia para a API de IA e retorna sugestões.
func WorkspaceTaskAssistantHandler(w http.ResponseWriter, r *http.Request) {
	// Este handler precisará do workspace_id (provavelmente da URL)
	// e da mensagem do usuário (do corpo da requisição).
	// vars := mux.Vars(r)
	// workspaceIDStr := vars["workspace_id"]
	// workspaceID, _ := strconv.ParseInt(workspaceIDStr, 10, 64)

	var userInput struct {
		UserMessage string `json:"user_message"`
		WorkspaceID int64  `json:"workspace_id"` // Frontend envia o ID do workspace atual
	}
	if err := json.NewDecoder(r.Body).Decode(&userInput); err != nil {
		utilities.LogError(err, "TaskAssistantHandler: Erro ao decodificar JSON")
		http.Error(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if userInput.UserMessage == "" || userInput.WorkspaceID == 0 {
		http.Error(w, `{"error": "Mensagem do usuário e ID do workspace são obrigatórios"}`, http.StatusBadRequest)
		return
	}

	// Aqui você chamaria a função que montamos para buscar o contexto do workspace:
	// workspaceContext, err := services.GetContextForIA(userInput.WorkspaceID, userInput.UserMessage)
	// if err != nil {
	//    utilities.LogError(err, "TaskAssistantHandler: Erro ao obter contexto do workspace")
	//    http.Error(w, `{"error": "Falha ao carregar contexto do workspace"}`, http.StatusInternalServerError)
	//    return
	// }
	// // O workspaceContext já inclui a userInput.UserMessage como MsgAoUsuario

	// Para este exemplo, vamos simular que o workspaceContext é o payload direto para a IA
	// (você pode precisar de um endpoint específico na IA para isso)
	// aiRequestPayload := workspaceContext

	// Mock: Simular o payload que a IA do assistente de tasks esperaria
	// Você precisa definir qual o endpoint da API de IA para o assistente de tasks
	// e qual o payload exato (que pode ser o workspaceContext ou uma variação).
	// Por agora, vamos assumir que é um endpoint "/task-assistant" e o payload é simples:
	type TaskAssistantPayload struct {
		Message     string `json:"message"`
		WorkspaceID int64  `json:"workspace_id"`
		// Inclua mais campos do contexto aqui, como nome do workspace, usuários, tarefas, etc.
		// conforme a struct IAWorkspaceContext.
	}
	aiRequestPayload := TaskAssistantPayload{Message: userInput.UserMessage, WorkspaceID: userInput.WorkspaceID}

	var aiResponse models.TaskAssistantAIResponse
	var aiErrorResponse models.TaskAssistantAIResponse

	statusCode, err := ai_service.CallAIAPI(r.Context(), "/task-assistant", aiRequestPayload, &aiResponse, &aiErrorResponse)
	if err != nil {
		utilities.LogError(err, fmt.Sprintf("TaskAssistantHandler: Erro ao chamar API de IA (status: %d)", statusCode))
		if aiErrorResponse.Error != "" {
			http.Error(w, fmt.Sprintf(`{"error": "Erro da IA: %s"}`, aiErrorResponse.Error), statusCode)
		} else {
			http.Error(w, `{"error": "Falha ao obter sugestões do assistente"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(aiResponse)
}
