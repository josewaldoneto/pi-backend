package models

import (
	"time"
	// Para firestore.ServerTimestamp, se usado diretamente na struct:
	// "cloud.google.com/go/firestore"
)

// UsuarioContext fornece detalhes do usuário para a IA
type UsuarioContext struct {
	Nome string `json:"nome"`
	Role string `json:"role,omitempty"`
}

// TarefaContext fornece detalhes da tarefa para a IA (versão simplificada para o prompt)
type TarefaContext struct {
	Titulo     string `json:"titulo"`
	Status     string `json:"status,omitempty"`
	Prioridade string `json:"prioridade,omitempty"`
	// Adicione outros campos se forem relevantes para a IA, como "descricao_curta"
}

// IAWorkspaceContext representa o JSON de contexto a ser enviado para a IA
type IAWorkspaceContext struct {
	WorkspaceIDStr string           `json:"workspace_id_str"` // ID do workspace (string)
	GrupoNome      string           `json:"grupo_nome"`
	DescricaoGrupo string           `json:"descricao_grupo"`
	Usuarios       []UsuarioContext `json:"usuarios"`
	Tarefas        []TarefaContext  `json:"tarefas_recentes"`     // Lista de tarefas relevantes
	MsgDoUsuario   string           `json:"msg_do_usuario_atual"` // O prompt/pergunta atual do usuário
}

// historico de requisições à IA
// AIRequestHistoryEntry representa um registro de requisição à IA no Firestore.
type AIRequestHistoryEntry struct {
	UserID        string    `firestore:"user_id"`         // Firebase UID do usuário que fez a requisição
	WorkspaceIDPg int64     `firestore:"workspace_id_pg"` // ID numérico do workspace no PostgreSQL
	AIServiceType string    `firestore:"ai_service_type"` // Ex: "code_review", "text_summary", "task_assistant"
	Timestamp     time.Time `firestore:"timestamp"`       // Data/Hora da requisição. O SDK Go converte para Timestamp do Firestore.
	// Alternativamente, use interface{} e atribua firestore.ServerTimestamp
	FrontendRequestPayload interface{} `firestore:"frontend_request_payload,omitempty"` // Payload original que o frontend enviou ao backend Go
	RequestToAI            interface{} `firestore:"request_to_ai"`                      // Payload que o backend Go enviou para a API Python de IA
	ResponseFromAI         interface{} `firestore:"response_from_ai,omitempty"`         // Payload que a API Python de IA retornou (em caso de sucesso)
	AIStatusCode           int         `firestore:"ai_status_code"`                     // Status HTTP retornado pela API de IA
	AIError                string      `firestore:"ai_error,omitempty"`                 // Mensagem de erro, se a chamada à API de IA falhou ou a IA retornou um erro
}

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

type TaskAssistantAIRequest struct {
	WorkspaceContext IAWorkspaceContext `json:"workspace_context"`
}
