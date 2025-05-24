package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"projeto-integrador/database"
	"projeto-integrador/firebase"
	"projeto-integrador/models"
	"strings"

	"firebase.google.com/go/v4/auth"
)

// AuthMiddleware é um middleware que verifica a autenticação
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Pega o token do header Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			LogError(fmt.Errorf("header de autorização ausente"), "Autenticação falhou")
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Verifica o token com Firebase
		verifiedToken, err := firebase.VerifyUserToken(tokenString)
		if err != nil {
			LogError(err, "Token inválido")
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Coloca o UID no contexto da requisição
		ctx := context.WithValue(r.Context(), "userUID", verifiedToken.UID)

		// Segue para o próximo handler
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// LoginHandler lida com o login do usuário
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var loginData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Criar um cliente HTTP
	client := &http.Client{}

	// Fazer a requisição para a API do Firebase
	firebaseURL := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", os.Getenv("FIREBASE_API_KEY"))

	reqBody := map[string]string{
		"email":             loginData.Email,
		"password":          loginData.Password,
		"returnSecureToken": "true",
	}

	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", firebaseURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Erro ao autenticar com Firebase", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Retornar o ID token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": result["idToken"].(string),
		"uid":   result["localId"].(string),
	})
}

// UserHandler retorna informações do usuário atual
func UserHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value("userUID").(string)

	db, err := database.ConnectPostgres()
	if err != nil {
		LogError(err, "Erro ao conectar ao banco de dados")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var user models.Usuario
	err = db.QueryRow("SELECT firebase_uid, email, display_name FROM users WHERE firebase_uid = $1", uid).
		Scan(&user.Firebase_uid, &user.Email, &user.DisplayName)
	if err != nil {
		LogError(err, "Erro ao buscar usuário")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateUserHandler atualiza informações do usuário
func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value("userUID").(string)

	var updateData struct {
		DisplayName string `json:"display_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		LogError(err, "Erro ao decodificar dados de atualização")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	db, err := database.ConnectPostgres()
	if err != nil {
		LogError(err, "Erro ao conectar ao banco de dados")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("UPDATE users SET display_name = $1 WHERE firebase_uid = $2",
		updateData.DisplayName, uid)
	if err != nil {
		LogError(err, "Erro ao atualizar usuário")
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User updated successfully",
	})
}

// GetUserHandler retorna informações de um usuário específico
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	db, err := database.ConnectPostgres()
	if err != nil {
		LogError(err, "Erro ao conectar ao banco de dados")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var user models.Usuario
	err = db.QueryRow("SELECT firebase_uid, email, display_name FROM users WHERE firebase_uid = $1", userID).
		Scan(&user.Firebase_uid, &user.Email, &user.DisplayName)
	if err != nil {
		LogError(err, "Erro ao buscar usuário")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetAllUsersHandler retorna todos os usuários
func GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	db, err := database.ConnectPostgres()
	if err != nil {
		LogError(err, "Erro ao conectar ao banco de dados")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT firebase_uid, email, display_name FROM users")
	if err != nil {
		LogError(err, "Erro ao buscar usuários")
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.Usuario
	for rows.Next() {
		var user models.Usuario
		if err := rows.Scan(&user.Firebase_uid, &user.Email, &user.DisplayName); err != nil {
			LogError(err, "Erro ao escanear usuário")
			continue
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	LogDebug("Iniciando registro de novo usuário")

	// Parse do corpo JSON
	var user models.Usuario
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		LogError(err, "Erro ao decodificar JSON do corpo da requisição")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validações
	if user.Email == "" {
		LogError(fmt.Errorf("email não fornecido"), "Validação falhou")
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	if user.Password == "" {
		LogError(fmt.Errorf("senha não fornecida"), "Validação falhou")
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	if user.DisplayName == "" {
		LogError(fmt.Errorf("nome de exibição não fornecido"), "Validação falhou")
		http.Error(w, "Display name is required", http.StatusBadRequest)
		return
	}

	// Verificar se o usuário já existe no Firebase pelo email
	ctx := context.Background()
	authClient := firebase.GetAuthClient()

	_, err = authClient.GetUserByEmail(ctx, user.Email)
	if err == nil {
		// Usuário já existe
		LogInfo("Tentativa de registro com email já existente: %s", user.Email)
		http.Error(w, "User already exists", http.StatusConflict)
		return
	} else if auth.IsUserNotFound(err) {
		// OK, usuário não existe, pode criar
		LogDebug("Criando novo usuário no Firebase: %s", user.Email)

		// Cria no Firebase
		params := (&auth.UserToCreate{}).
			Email(user.Email).
			Password(user.Password).
			DisplayName(user.DisplayName).
			Disabled(false)

		firebaseUser, createErr := authClient.CreateUser(ctx, params)
		if createErr != nil {
			LogError(createErr, "Erro ao criar usuário no Firebase")
			http.Error(w, "Failed to create user in Firebase", http.StatusInternalServerError)
			return
		}

		// Agora salva no PostgreSQL
		db, err := database.ConnectPostgres()
		if err != nil {
			LogError(err, "Erro ao conectar ao banco de dados")
			http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
			return
		}

		_, insertErr := db.Exec(
			"INSERT INTO users (firebase_uid, email, display_name) VALUES ($1, $2, $3)",
			firebaseUser.UID, user.Email, user.DisplayName,
		)
		if insertErr != nil {
			LogError(insertErr, "Erro ao salvar usuário no banco de dados")
			http.Error(w, "Failed to save user in database", http.StatusInternalServerError)
			return
		}
		defer db.Close()

		// Gerar Custom Token para o Frontend
		customToken, tokenErr := authClient.CustomToken(ctx, firebaseUser.UID)
		if tokenErr != nil {
			LogError(tokenErr, "Erro ao gerar custom token")
			http.Error(w, "Failed to generate authentication token", http.StatusInternalServerError)
			return
		}

		LogInfo("Usuário registrado com sucesso: %s", user.Email)

		// Resposta de sucesso para o frontend
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message":     "User created successfully and ready to sign in",
			"uid":         firebaseUser.UID,
			"customToken": customToken,
		})
		return
	} else {
		// Outro erro inesperado
		LogError(err, "Erro inesperado ao verificar usuário no Firebase")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Suponha que o UID venha do contexto (middleware de autenticação)
	uid := r.Context().Value("userUID")
	if uid == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Obter o client do Firebase
	ctx := context.Background()
	authClient := firebase.GetAuthClient()

	// Revogar os tokens de refresh do usuário
	err := authClient.RevokeRefreshTokens(ctx, uid.(string))
	if err != nil {
		log.Printf("Erro ao revogar tokens: %v", err)
		http.Error(w, "Erro ao fazer logout", http.StatusInternalServerError)
		return
	}

	log.Printf("Tokens revogados para UID: %s", uid)

	// Opcional: remover cookies/sessões locais, se estiver usando cookies HTTP-only
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // expira imediatamente
	})

	// Retorna sucesso
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logout efetuado com sucesso",
	})
}

func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	// Handle delete user logic
}

func SocialLoginHandler(w http.ResponseWriter, r *http.Request) {
	// Handle social login logic
}

func main() {
	// Inicializar conexão com o banco de dados
	db, err := sql.Open("postgres", "sua_string_de_conexao")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Inicializar handlers
	InitDB(db)
	err = InitFirebase("caminho/para/suas/credenciais.json")
	if err != nil {
		log.Fatal(err)
	}

	// Configurar rotas e iniciar servidor
	// ...
}
