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

	// --- Rotas Públicas ---
	r.HandleFunc("/register", handlers.RegisterHandler).Methods("POST")
	// r.HandleFunc("/login", handlers.LoginHandler).Methods("POST") // Seu handler de login tradicional (se existir)
	// Adicionando o handler para finalizar login com token Firebase (social, etc.)
	r.HandleFunc("/auth/finalize-login", handlers.FinalizeFirebaseLoginHandler).Methods("POST")

	// --- Rotas Autenticadas ---

	// Logout - idealmente autenticado se realiza ações no backend (ex: revogar tokens)
	r.HandleFunc("/logout", handlers.AuthMiddleware(handlers.LogoutHandler)).Methods("POST")

	// Rotas de Usuário (protegidas)
	r.HandleFunc("/user", handlers.AuthMiddleware(handlers.UserHandler)).Methods("GET") //ok
	r.HandleFunc("/user", handlers.AuthMiddleware(handlers.UpdateUserHandler)).Methods("PUT")
	r.HandleFunc("/user", handlers.AuthMiddleware(handlers.DeleteUserHandler)).Methods("DELETE")
	r.HandleFunc("/users", handlers.AuthMiddleware(handlers.GetAllUsersHandler)).Methods("GET")  //ok
	r.HandleFunc("/users/{id}", handlers.AuthMiddleware(handlers.GetUserHandler)).Methods("GET") //nao ta funcionando a busca do parametro na rota

	// Novas Rotas de Workspace (protegidas)
	r.HandleFunc("/workspaces", handlers.AuthMiddleware(handlers.CreateWorkspaceHandler)).Methods("POST")
	r.HandleFunc("/workspaces/{workspace_id}", handlers.AuthMiddleware(handlers.GetWorkspaceInfoHandler)).Methods("GET")
	r.HandleFunc("/workspaces/{workspace_id}", handlers.AuthMiddleware(handlers.UpdateWorkspaceHandler)).Methods("PUT")
	r.HandleFunc("/workspaces/{workspace_id}", handlers.AuthMiddleware(handlers.DeleteWorkspaceHandler)).Methods("DELETE")
	r.HandleFunc("/workspaces/{workspace_id}/members", handlers.AuthMiddleware(handlers.ListWorkspaceMembersHandler)).Methods("GET")
	r.HandleFunc("/workspaces/{workspace_id}/members", handlers.AuthMiddleware(handlers.AddUserToWorkspaceHandler)).Methods("POST")
	r.HandleFunc("/workspaces/{workspace_id}/members/{member_firebase_uid}", handlers.AuthMiddleware(handlers.RemoveUserFromWorkspaceHandler)).Methods("DELETE")

	// Novas Rotas de Tarefas (protegidas e aninhadas sob workspaces)
	r.HandleFunc("/workspaces/{workspace_id}/tasks", handlers.AuthMiddleware(handlers.CreateTaskHandler)).Methods("POST")
	r.HandleFunc("/workspaces/{workspace_id}/tasks", handlers.AuthMiddleware(handlers.ListTasksHandler)).Methods("GET")
	r.HandleFunc("/workspaces/{workspace_id}/tasks/{task_doc_id}", handlers.AuthMiddleware(handlers.GetTaskHandler)).Methods("GET")
	r.HandleFunc("/workspaces/{workspace_id}/tasks/{task_doc_id}", handlers.AuthMiddleware(handlers.UpdateTaskHandler)).Methods("PUT")
	r.HandleFunc("/workspaces/{workspace_id}/tasks/{task_doc_id}", handlers.AuthMiddleware(handlers.DeleteTaskHandler)).Methods("DELETE")

	// Configuração do CORS
	headers := gorillahandlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	methods := gorillahandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})

	// Obter origens permitidas das variáveis de ambiente
	allowedOriginsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsEnv == "" {
		allowedOrigins = []string{"*"} // Fallback para permitir todas as origens se não estiver configurado
		utilities.LogInfo("CORS_ALLOWED_ORIGINS não definida, permitindo todas as origens ('*'). Defina para maior segurança em produção.")
	} else {
		allowedOrigins = strings.Split(allowedOriginsEnv, ",")
	}
	origins := gorillahandlers.AllowedOrigins(allowedOrigins)
	utilities.LogInfo("Configurando CORS com origens permitidas: %v", allowedOrigins)

	// Aplicar middleware CORS e iniciar servidor
	handler := gorillahandlers.CORS(headers, methods, origins)(r)

	// Obter porta do servidor das variáveis de ambiente
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080" // Porta padrão se não estiver configurada
	}

	utilities.LogInfo("Servidor iniciado na porta %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
