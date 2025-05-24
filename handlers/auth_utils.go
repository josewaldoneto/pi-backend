package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

var db *sql.DB
var firebaseApp *firebase.App

// InitDB inicializa a conexão com o banco de dados
func InitDB(database *sql.DB) {
	LogInfo("Inicializando conexão com o banco de dados")
	db = database
}

// InitFirebase inicializa a conexão com o Firebase
func InitFirebase(credentialsFile string) error {
	LogInfo("Inicializando conexão com o Firebase usando arquivo: %s", credentialsFile)
	opt := option.WithCredentialsFile(credentialsFile)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		LogError(err, "Erro ao inicializar Firebase")
		return err
	}
	firebaseApp = app
	LogInfo("Conexão com Firebase estabelecida com sucesso")
	return nil
}

// getUIDFromToken extrai o UID do usuário do token Firebase
func getUIDFromToken(r *http.Request) (string, error) {
	LogDebug("Extraindo UID do token Firebase")
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		LogError(fmt.Errorf("token não fornecido"), "Falha na autenticação")
		return "", fmt.Errorf("token não fornecido")
	}

	token := strings.Replace(authHeader, "Bearer ", "", 1)
	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		LogError(err, "Erro ao obter cliente de autenticação do Firebase")
		return "", err
	}

	tokenInfo, err := client.VerifyIDToken(context.Background(), token)
	if err != nil {
		LogError(err, "Erro ao verificar token do Firebase")
		return "", err
	}

	LogDebug("Token verificado com sucesso para UID: %s", tokenInfo.UID)
	return tokenInfo.UID, nil
}

// isWorkspaceMember verifica se um usuário é membro de um workspace
func isWorkspaceMember(uid string, workspaceID string) (bool, error) {
	LogDebug("Verificando se usuário %s é membro do workspace %s", uid, workspaceID)
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM workspace_members wm
			JOIN users u ON wm.user_id = u.id
			WHERE u.firebase_uid = $1 AND wm.workspace_id = $2
		)
	`
	err := db.QueryRow(query, uid, workspaceID).Scan(&exists)
	if err != nil {
		LogError(err, "Erro ao verificar membro do workspace")
		return false, err
	}

	if exists {
		LogDebug("Usuário %s é membro do workspace %s", uid, workspaceID)
	} else {
		LogDebug("Usuário %s não é membro do workspace %s", uid, workspaceID)
	}
	return exists, nil
}
