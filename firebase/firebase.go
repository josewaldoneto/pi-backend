package firebase

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

func InitializeFirebase() (*firebase.App, error) {
	credentialsPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	if credentialsPath == "" {
		return nil, fmt.Errorf("FIREBASE_CREDENTIALS_PATH não está definido nas variáveis de ambiente")
	}

	opt := option.WithCredentialsFile(credentialsPath)

	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Erro ao inicializar Firebase: %v", err)
	}

	fmt.Println("Firebase inicializado com sucesso!")
	return app, nil
}

// retorna o cliente de autenticação
func GetAuthClient() *auth.Client {
	ctx := context.Background()
	app, err := InitializeFirebase()
	if err != nil {
		log.Fatalf("Erro ao inicializar Firebase: %v", err)
	}
	// Obter o cliente de autenticação
	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("Erro ao obter cliente de Auth: %v", err)
	}
	return authClient
}

func GetFirestoreClient() (*firestore.Client, error) {
	app, err := InitializeFirebase()
	if err != nil {
		// Retorne o erro em vez de Fatalf
		return nil, fmt.Errorf("erro ao inicializar Firebase: %w", err)
	}
	ctx := context.Background()
	// Obter o cliente do Firestore a partir do app
	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		// Retorne o erro em vez de Fatalf
		return nil, fmt.Errorf("erro ao obter cliente do Firestore: %w", err)
	}
	return firestoreClient, nil
}
