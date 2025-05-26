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

	// Aplicar o middleware de logging em todas as rotas
	r.Use(handlers.LoggingMiddleware)

	// Rotas públicas
	r.HandleFunc("/register", handlers.RegisterHandler).Methods("POST")
	// r.HandleFunc("/login", handlers.LoginHandler).Methods("POST")

	r.HandleFunc("/logout", handlers.LogoutHandler).Methods("POST")
	// Rotas protegidas
	r.HandleFunc("/user", handlers.AuthMiddleware(handlers.UserHandler)).Methods("GET")
	r.HandleFunc("/user", handlers.AuthMiddleware(handlers.UpdateUserHandler)).Methods("PUT")
	r.HandleFunc("/user", handlers.AuthMiddleware(handlers.DeleteUserHandler)).Methods("DELETE")
	r.HandleFunc("/users", handlers.AuthMiddleware(handlers.GetAllUsersHandler)).Methods("GET")
	r.HandleFunc("/users/{id}", handlers.AuthMiddleware(handlers.GetUserHandler)).Methods("GET")

	// Configuração do CORS
	headers := gorillahandlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	methods := gorillahandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})

	// Obter origens permitidas das variáveis de ambiente
	allowedOrigins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"} // Fallback para permitir todas as origens se não estiver configurado
	}
	origins := gorillahandlers.AllowedOrigins(allowedOrigins)

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
