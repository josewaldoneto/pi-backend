package main

import (
	"log"
	"net/http"
	"os"
	"projeto-integrador/handlers"  // Seus handlers, incluindo os de workspace
	"projeto-integrador/utilities" // Seu pacote de utilitários para logs
	"strings"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func LoadRoutes() {
	// Inicializar o sistema de logs
	utilities.InitLogger() //

	r := mux.NewRouter()

	// Aplicar o middleware de logging global em todas as rotas
	r.Use(handlers.LoggingMiddleware) //

	// --- Rotas Públicas ---
	r.HandleFunc("/register", handlers.RegisterHandler).Methods("POST") //
	// r.HandleFunc("/login", handlers.LoginHandler).Methods("POST") // // Seu handler de login tradicional (se houver)
	r.HandleFunc("/auth/finalize-login", handlers.FinalizeFirebaseLoginHandler).Methods("POST") // Handler para processar token do Firebase (social ou email/senha do cliente)

	// --- Rotas Autenticadas ---

	// Logout (precisa de autenticação para saber quem deslogar no backend, ex: invalidar tokens de refresh)
	// Se o logout for apenas client-side, pode não precisar de AuthMiddleware.
	// Mas se o backend fizer algo (ex: revogar refresh tokens do Firebase), precisa.
	r.HandleFunc("/logout", handlers.AuthMiddleware(handlers.LogoutHandler)).Methods("POST") // Ajustado para ser autenticado

	// Opção 1: Usando Subrouter com .Use() (requer AuthMiddleware como func(http.Handler) http.Handler)
	// workspaceRouter := r.PathPrefix("/workspaces").Subrouter()
	// workspaceRouter.Use(handlers.AuthMiddleware) // Este AuthMiddleware deve ser func(http.Handler) http.Handler
	// workspaceRouter.HandleFunc("", handlers.CreateWorkspaceHandler).Methods("POST")
	// workspaceRouter.HandleFunc("/{workspace_id}", handlers.GetWorkspaceInfoHandler).Methods("GET")
	// workspaceRouter.HandleFunc("/{workspace_id}", handlers.UpdateWorkspaceHandler).Methods("PUT")
	// workspaceRouter.HandleFunc("/{workspace_id}", handlers.DeleteWorkspaceHandler).Methods("DELETE")
	// workspaceRouter.HandleFunc("/{workspace_id}/members", handlers.ListWorkspaceMembersHandler).Methods("GET")
	// workspaceRouter.HandleFunc("/{workspace_id}/members", handlers.AddUserToWorkspaceHandler).Methods("POST")
	// workspaceRouter.HandleFunc("/{workspace_id}/members/{member_firebase_uid}", handlers.RemoveUserFromWorkspaceHandler).Methods("DELETE")

	// Opção 2: Seguindo seu padrão atual de envolver cada handler individualmente
	// (Isto funciona se AuthMiddleware for func(http.HandlerFunc) http.HandlerFunc)
	r.HandleFunc("/workspaces", handlers.AuthMiddleware(handlers.CreateWorkspaceHandler)).Methods("POST")
	r.HandleFunc("/workspaces/{workspace_id}", handlers.AuthMiddleware(handlers.GetWorkspaceInfoHandler)).Methods("GET")
	r.HandleFunc("/workspaces/{workspace_id}", handlers.AuthMiddleware(handlers.UpdateWorkspaceHandler)).Methods("PUT")
	r.HandleFunc("/workspaces/{workspace_id}", handlers.AuthMiddleware(handlers.DeleteWorkspaceHandler)).Methods("DELETE")
	r.HandleFunc("/workspaces/{workspace_id}/members", handlers.AuthMiddleware(handlers.ListWorkspaceMembersHandler)).Methods("GET")
	r.HandleFunc("/workspaces/{workspace_id}/members", handlers.AuthMiddleware(handlers.AddUserToWorkspaceHandler)).Methods("POST")
	r.HandleFunc("/workspaces/{workspace_id}/members/{member_firebase_uid}", handlers.AuthMiddleware(handlers.RemoveUserFromWorkspaceHandler)).Methods("DELETE")

	// Configuração do CORS
	headers := gorillahandlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}) //
	methods := gorillahandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})           //

	allowedOriginsEnv := os.Getenv("CORS_ALLOWED_ORIGINS") //
	var allowedOrigins []string
	if allowedOriginsEnv == "" {
		allowedOrigins = []string{"*"} // Fallback

	} else {
		allowedOrigins = strings.Split(allowedOriginsEnv, ",") //
	}
	origins := gorillahandlers.AllowedOrigins(allowedOrigins) //

	utilities.LogInfo("Configurando CORS com origens permitidas: %v", allowedOrigins)

	handler := gorillahandlers.CORS(headers, methods, origins)(r) //

	port := os.Getenv("SERVER_PORT") //
	if port == "" {
		port = "8080" // Porta padrão
	}

	utilities.LogInfo("Servidor iniciado na porta %s", port) //
	log.Fatal(http.ListenAndServe(":"+port, handler))        //
}
