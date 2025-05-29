package models

// Para Code Review
type CodeReviewAIRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

type CodeReviewAIResponse struct {
	Review string `json:"review,omitempty"`
	Error  string `json:"error,omitempty"` // Para capturar erros da API de IA
}

// Para Resumo de Texto
type SummarizeTextAIRequest struct {
	Text string `json:"text"`
}

type SummarizeTextAIResponse struct {
	Summary string `json:"summary,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Para Geração de Ideias para Mapa Mental
type MindMapIdeasAIRequest struct {
	Text string `json:"text"`
}

type MindMapIdeasAIResponse struct {
	MindMapIdeas string `json:"mind_map_ideas,omitempty"` // Ou poderia ser uma estrutura mais complexa
	Error        string `json:"error,omitempty"`
}

// Para o Assistente de Tarefas do Workspace (usando o contexto que definimos antes)
// Supondo que IAWorkspaceContext já está definido em outro lugar (ex: services ou models)
// type TaskAssistantAIRequest models.IAWorkspaceContext // Se for exatamente o mesmo
type TaskAssistantAIResponse struct {
	Suggestions []string `json:"suggestions,omitempty"` // Exemplo
	Error       string   `json:"error,omitempty"`
}
