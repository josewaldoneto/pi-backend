package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"projeto-integrador/database"
	"projeto-integrador/firebase"
	"projeto-integrador/models"
	"projeto-integrador/utilities"
	"strconv"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gorilla/mux"
)

type SocialLoginInput struct {
	IDToken string `json:"idToken"`
}

// SocialLoginResponse define a estrutura da resposta de sucesso
type SocialLoginResponse struct {
	Message     string `json:"message"`
	FirebaseUID string `json:"firebaseUid"` // UID do Firebase, que é o firebase_uid no seu banco
	// Você pode adicionar mais campos aqui, como um token de sessão da sua aplicação, se aplicável
}

// AuthMiddleware é um middleware que verifica a autenticação
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Pega o token do header Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utilities.LogError(fmt.Errorf("header de autorização ausente"), "Autenticação falhou")
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Verifica o token com Firebase
		verifiedToken, err := firebase.VerifyUserToken(tokenString)
		if err != nil {
			utilities.LogError(err, "Token inválido")
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Coloca o UID no contexto da requisição
		ctx := context.WithValue(r.Context(), "userUID", verifiedToken.UID)

		// Segue para o próximo handler
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// FinalizeFirebaseLoginHandler processa um ID Token do Firebase (de login social ou outro)
// para verificar o usuário e sincronizá-lo com o banco de dados local.
func FinalizeFirebaseLoginHandler(w http.ResponseWriter, r *http.Request) {
	utilities.LogInfo("Recebida requisição para finalizar login com ID Token do Firebase.")

	var input SocialLoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utilities.LogError(err, "Erro ao decodificar corpo da requisição para finalizar login Firebase")
		http.Error(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if strings.TrimSpace(input.IDToken) == "" {
		utilities.LogError(nil, "ID Token não fornecido no corpo da requisição")
		http.Error(w, "ID Token é obrigatório", http.StatusBadRequest)
		return
	}

	// 1. Verificar o ID Token com o Firebase
	// Log apenas uma parte do token por segurança, se necessário
	tokenLoggablePart := input.IDToken
	if len(tokenLoggablePart) > 15 {
		tokenLoggablePart = tokenLoggablePart[:15] + "..."
	}
	utilities.LogDebug("Verificando ID Token do Firebase: %s", tokenLoggablePart)

	verifiedToken, err := firebase.VerifyUserToken(input.IDToken)
	if err != nil {
		utilities.LogError(err, "Falha ao verificar ID Token do Firebase")
		// Não exponha muitos detalhes do erro ao cliente por segurança
		http.Error(w, "Token inválido ou falha na verificação", http.StatusUnauthorized)
		return
	}
	utilities.LogInfo("ID Token verificado com sucesso para Firebase UID: %s", verifiedToken.UID)

	// 2. Conectar ao banco de dados
	// Se você usa uma variável db global inicializada em auth_utils.go, use-a aqui.
	// Caso contrário, conecte como em outros handlers:
	dbConn, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "Erro ao conectar ao banco de dados para finalizar login Firebase")
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}
	defer dbConn.Close()

	// 3. Verificar/Criar usuário no banco de dados PostgreSQL
	// A função firebase.CheckOrCreateUserInPostgres recebe (db *sql.DB, token *auth.Token)
	// e retorna o UID do banco (que deve ser o mesmo Firebase UID).
	utilities.LogDebug("Sincronizando usuário com banco de dados local para Firebase UID: %s", verifiedToken.UID)
	localUserUID, err := firebase.CheckOrCreateUserInPostgres(dbConn, verifiedToken)
	if err != nil {
		utilities.LogError(err, "Erro ao sincronizar usuário com banco de dados local")
		http.Error(w, "Erro interno do servidor ao processar usuário", http.StatusInternalServerError)
		return
	}
	utilities.LogInfo("Usuário (Firebase UID: %s) sincronizado com sucesso no banco de dados local (ID local: %s).", verifiedToken.UID, localUserUID)

	// 4. Responder com sucesso
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SocialLoginResponse{
		Message:     "Login finalizado e usuário sincronizado com sucesso.",
		FirebaseUID: localUserUID, // Este é o UID que está no seu banco local (deve ser igual ao verifiedToken.UID)
	})
}

// UserHandler retorna informações do usuário atual
func UserHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value("userUID").(string)

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "Erro ao conectar ao banco de dados")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var user models.Usuario
	err = db.QueryRow("SELECT firebase_uid, email, display_name FROM users WHERE firebase_uid = $1", uid).
		Scan(&user.Firebase_uid, &user.Email, &user.DisplayName)
	if err != nil {
		utilities.LogError(err, "Erro ao buscar usuário")
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
		utilities.LogError(err, "Erro ao decodificar dados de atualização")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "Erro ao conectar ao banco de dados")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("UPDATE users SET display_name = $1 WHERE firebase_uid = $2",
		updateData.DisplayName, uid)
	if err != nil {
		utilities.LogError(err, "Erro ao atualizar usuário")
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
	vars := mux.Vars(r)  // Pega as variáveis do caminho da URL
	userID := vars["id"] // Acessa a variável "id" definida na rota

	if userID == "" {
		// Este caso não deveria acontecer se a rota tem {id} e mux está funcionando,
		// mas é uma verificação de segurança. O erro que você viu ("User ID is required")
		// aconteceria se 'vars["id"]' não existisse ou estivesse vazio.
		utilities.LogError(fmt.Errorf("ID do usuário ausente nos parâmetros da rota"), "GetUserHandler")
		http.Error(w, "User ID is required in path", http.StatusBadRequest)
		return
	}

	userIDInt, errConv := strconv.ParseInt(userID, 10, 64)
	if errConv != nil {
		utilities.LogError(errConv, "GetUserHandler: ID do usuário inválido na rota")
		http.Error(w, "Invalid User ID format", http.StatusBadRequest)
		return
	}

	db, err := database.ConnectPostgres()
	if err != nil {
		utilities.LogError(err, "Erro ao conectar ao banco de dados")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var user models.Usuario
	err = db.QueryRow("SELECT firebase_uid, email, display_name FROM users WHERE id = $1", userIDInt).
		Scan(&user.Firebase_uid, &user.Email, &user.DisplayName)
	if err != nil {
		utilities.LogError(err, "Erro ao buscar usuário")
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
		utilities.LogError(err, "Erro ao conectar ao banco de dados")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT firebase_uid, email, display_name FROM users")
	if err != nil {
		utilities.LogError(err, "Erro ao buscar usuários")
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.Usuario
	for rows.Next() {
		var user models.Usuario
		if err := rows.Scan(&user.Firebase_uid, &user.Email, &user.DisplayName); err != nil {
			utilities.LogError(err, "Erro ao escanear usuário")
			continue
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	utilities.LogDebug("Iniciando registro de novo usuário")

	// Parse do corpo JSON
	var user models.Usuario
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		utilities.LogError(err, "Erro ao decodificar JSON do corpo da requisição")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validações
	if user.Email == "" {
		utilities.LogError(fmt.Errorf("email não fornecido"), "Validação falhou")
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	if user.Password == "" {
		utilities.LogError(fmt.Errorf("senha não fornecida"), "Validação falhou")
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	if user.DisplayName == "" {
		utilities.LogError(fmt.Errorf("nome de exibição não fornecido"), "Validação falhou")
		http.Error(w, "Display name is required", http.StatusBadRequest)
		return
	}

	// Verificar se o usuário já existe no Firebase pelo email
	ctx := context.Background()
	authClient := firebase.GetAuthClient()

	_, err = authClient.GetUserByEmail(ctx, user.Email)
	if err == nil {
		// Usuário já existe
		utilities.LogInfo("Tentativa de registro com email já existente: %s", user.Email)
		http.Error(w, "User already exists", http.StatusConflict)
		return
	} else if auth.IsUserNotFound(err) {
		// OK, usuário não existe, pode criar
		utilities.LogDebug("Criando novo usuário no Firebase: %s", user.Email)

		// Cria no Firebase
		params := (&auth.UserToCreate{}).
			Email(user.Email).
			Password(user.Password).
			DisplayName(user.DisplayName).
			Disabled(false)

		firebaseUser, createErr := authClient.CreateUser(ctx, params)
		if createErr != nil {
			utilities.LogError(createErr, "Erro ao criar usuário no Firebase")
			http.Error(w, "Failed to create user in Firebase", http.StatusInternalServerError)
			return
		}

		// Agora salva no PostgreSQL
		db, err := database.ConnectPostgres()
		if err != nil {
			utilities.LogError(err, "Erro ao conectar ao banco de dados")
			http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
			return
		}

		_, insertErr := db.Exec(
			"INSERT INTO users (firebase_uid, email, display_name) VALUES ($1, $2, $3)",
			firebaseUser.UID, user.Email, user.DisplayName,
		)
		if insertErr != nil {
			utilities.LogError(insertErr, "Erro ao salvar usuário no banco de dados")
			http.Error(w, "Failed to save user in database", http.StatusInternalServerError)
			return
		}
		// 4. Criar Workspace Privado
		// A função CreatePrivateWorkspace agora retorna (*Workspace, error)
		// Usaremos a variável 'db' que já está aberta.
		errWorkspace := models.CreatePrivateWorkspace(db, firebaseUser.UID)
		if errWorkspace != nil {
			utilities.LogError(errWorkspace, "Falha ao criar workspace privado para o usuário "+firebaseUser.UID)

			// Iniciar Rollback:
			// a. Deletar usuário do PostgreSQL
			utilities.LogInfo("Tentando reverter inserção do usuário no PostgreSQL UID: %s", firebaseUser.UID)
			_, dbDeleteErr := db.Exec("DELETE FROM users WHERE firebase_uid = $1", firebaseUser.UID)
			if dbDeleteErr != nil {
				utilities.LogError(dbDeleteErr, "Falha CRÍTICA ao tentar reverter inserção do usuário no PostgreSQL UID: "+firebaseUser.UID)
				// O usuário pode permanecer no DB e no Firebase, mas sem workspace.
			} else {
				utilities.LogInfo("Usuário removido do PostgreSQL com sucesso (Rollback): %s", firebaseUser.UID)
			}

			// b. Deletar usuário do Firebase
			utilities.LogInfo("Tentando reverter criação do usuário no Firebase UID: %s", firebaseUser.UID)
			if delErr := firebase.DeleteUser(firebaseUser.UID); delErr != nil {
				utilities.LogError(delErr, "Falha CRÍTICA ao tentar reverter criação do usuário no Firebase UID: "+firebaseUser.UID)
				// O usuário pode permanecer no Firebase, mesmo que tenha sido removido do DB local.
			} else {
				utilities.LogInfo("Usuário removido do Firebase com sucesso (Rollback): %s", firebaseUser.UID)
			}

			http.Error(w, "Erro interno do servidor ao finalizar configuração do usuário", http.StatusInternalServerError)
			return // Interrompe o fluxo de registro
		}
		utilities.LogInfo("Workspace privado criado com sucesso para Firebase UID: %s", firebaseUser.UID)

		defer db.Close()
		// Gerar Custom Token para o Frontend
		customToken, tokenErr := authClient.CustomToken(ctx, firebaseUser.UID)
		if tokenErr != nil {
			utilities.LogError(tokenErr, "Erro ao gerar custom token")
			http.Error(w, "Failed to generate authentication token", http.StatusInternalServerError)
			return
		}

		utilities.LogInfo("Usuário registrado com sucesso: %s", user.Email)

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
		utilities.LogError(err, "Erro inesperado ao verificar usuário no Firebase")
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
