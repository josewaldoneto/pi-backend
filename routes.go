package main

import (
	"log"
	"net/http"
	"os"
	"projeto-integrador/handlers"
	"projeto-integrador/utilities"
	"strings"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func LoadRoutes() {
	// Inicializar o sistema de logs
	utilities.InitLogger()

	r := mux.NewRouter()

	// Aplicar o middleware de logging global em todas as rotas
	r.Use(handlers.LoggingMiddleware)

	// --- Rotas de Autenticação e Públicas ---
	r.HandleFunc("/auth/register", handlers.RegisterHandler).Methods("POST")                      //ok
	r.HandleFunc("/auth/finalize-login", handlers.FinalizeFirebaseLoginHandler).Methods("POST")   //ok
	r.HandleFunc("/auth/logout", handlers.AuthMiddleware(handlers.LogoutHandler)).Methods("POST") //ok
	// --- Rotas de Usuário (autenticado, referindo-se ao próprio usuário logado) ---
	r.HandleFunc("/user/info", handlers.AuthMiddleware(handlers.UserHandler)).Methods("GET")            //ok
	r.HandleFunc("/user/update", handlers.AuthMiddleware(handlers.UpdateUserHandler)).Methods("PUT")    //ok
	r.HandleFunc("/user/delete", handlers.AuthMiddleware(handlers.DeleteUserHandler)).Methods("DELETE") // precisa ser feito
	// --- Rotas de Usuários (operações gerais, protegidas) ---
	r.HandleFunc("/users/list", handlers.AuthMiddleware(handlers.GetAllUsersHandler)).Methods("GET")                     //ok
	r.HandleFunc("/users/info/{id}", handlers.AuthMiddleware(handlers.GetUserHandler)).Methods("GET")                    //ok
	r.HandleFunc("/user/my-workspaces/list", handlers.AuthMiddleware(handlers.ListUserWorkspacesHandler)).Methods("GET") //ok

	// --- Rotas de Workspace (protegidas) ---
	r.HandleFunc("/workspace/create", handlers.AuthMiddleware(handlers.CreateWorkspaceHandler)).Methods("POST")                                  //ok
	r.HandleFunc("/workspace/info/{workspace_id}", handlers.AuthMiddleware(handlers.GetWorkspaceInfoHandler)).Methods("GET")                     //ok
	r.HandleFunc("/workspace/update/{workspace_id}", handlers.AuthMiddleware(handlers.UpdateWorkspaceHandler)).Methods("PUT")                    //ok
	r.HandleFunc("/workspace/delete/{workspace_id}", handlers.AuthMiddleware(handlers.DeleteWorkspaceHandler)).Methods("DELETE")                 //ok
	r.HandleFunc("/workspace/{workspace_id}/members/list", handlers.AuthMiddleware(handlers.ListWorkspaceMembersHandler)).Methods("GET")         //ok
	r.HandleFunc("/workspace/{workspace_id}/members/add", handlers.AuthMiddleware(handlers.AddUserToWorkspaceHandler)).Methods("POST")           //ok
	r.HandleFunc("/workspace/{workspace_id}/members/remove", handlers.AuthMiddleware(handlers.RemoveUserFromWorkspaceHandler)).Methods("DELETE") //ok

	// --- Rotas de Tarefas (protegidas e aninhadas sob workspaces) ---
	r.HandleFunc("/workspace/{workspace_id}/task/create", handlers.AuthMiddleware(handlers.CreateTaskHandler)).Methods("POST")                 //ok
	r.HandleFunc("/workspace/{workspace_id}/task/list", handlers.AuthMiddleware(handlers.ListTasksHandler)).Methods("GET")                     //ok
	r.HandleFunc("/workspace/{workspace_id}/task/info/{task_doc_id}", handlers.AuthMiddleware(handlers.GetTaskHandler)).Methods("GET")         //ok
	r.HandleFunc("/workspace/{workspace_id}/task/update/{task_doc_id}", handlers.AuthMiddleware(handlers.UpdateTaskHandler)).Methods("PUT")    //ok
	r.HandleFunc("/workspace/{workspace_id}/task/delete/{task_doc_id}", handlers.AuthMiddleware(handlers.DeleteTaskHandler)).Methods("DELETE") //ok

	// --- Rotas para Funcionalidades de IA (protegidas) ---
	r.HandleFunc("/workspace/{workspace_id}/ai/summarize-text", handlers.AuthMiddleware(handlers.SummarizeTextAIHandler)).Methods("POST")
	r.HandleFunc("/workspace/{workspace_id}/ai/code-review", handlers.AuthMiddleware(handlers.CodeReviewAIHandler)).Methods("POST")
	r.HandleFunc("/workspace/{workspace_id}/ai/mindmap-ideas", handlers.AuthMiddleware(handlers.GenerateMindMapIdeasAIHandler)).Methods("POST")
	r.HandleFunc("/workspace/{workspace_id}/ai/task-assistant", handlers.AuthMiddleware(handlers.WorkspaceTaskAssistantHandler)).Methods("POST")
	// Opção B: Com workspace_id na rota (handler precisaria ser ajustado para ler da rota e do corpo)
	// r.HandleFunc("/workspace/{workspace_id}/ai/task-assistant", handlers.AuthMiddleware(handlers.WorkspaceTaskAssistantHandler)).Methods("POST")

	// ... (resto do seu routes.go) ...

	// Configuração do CORS
	headers := gorillahandlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	methods := gorillahandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})

	allowedOriginsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsEnv == "" {
		allowedOrigins = []string{"*"}
		utilities.LogInfo("CORS_ALLOWED_ORIGINS não definida, permitindo todas as origens ('*'). Defina para maior segurança em produção.")
	} else {
		allowedOrigins = strings.Split(allowedOriginsEnv, ",")
	}
	origins := gorillahandlers.AllowedOrigins(allowedOrigins)
	utilities.LogInfo("Configurando CORS com origens permitidas: %v", allowedOrigins)

	handler := gorillahandlers.CORS(headers, methods, origins)(r)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	utilities.LogInfo("Servidor iniciado na porta %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
