package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func ConnectPostgres() (*sql.DB, error) {
	// Configurações do banco de dados
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	// Monta a string de conexãos
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	// Abre a conexão
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Erro ao abrir conexão com o banco de dados: %v", err)
		return nil, err
	}

	// Testa a conexão
	err = db.Ping()
	if err != nil {
		log.Printf("Erro ao conectar ao banco de dados: %v", err)
		return nil, err
	}

	log.Println("Conectado ao PostgreSQL com sucesso!")
	return db, nil
}
