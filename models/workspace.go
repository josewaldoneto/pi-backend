package models

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

type Workspace struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"is_public"`
	OwnerUID    string    `json:"owner_uid"` // Firebase UID do dono
	CreatedAt   time.Time `json:"created_at"`
	Members     int       `json:"members"`
}

type WorkspaceInvite struct {
	ID          int64      `json:"id"`
	WorkspaceID int64      `json:"workspace_id"`
	InviteCode  string     `json:"invite_code"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"` // ponteiro permite nulo
	Role        string     `json:"role"`
}

// Relação Usuário → Workspace

type UserWorkspace struct {
	WorkspaceID int64     `json:"workspace_id"`
	UserID      int64     `json:"user_id"` // Alterado para int64 para corresponder a users.id
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

// UserWorkspaceInfo descreve um workspace do qual o usuário é membro.
type UserWorkspaceInfo struct {
	ID       int64  `json:"id"`        // ID do workspace
	Name     string `json:"name"`      // Nome do workspace
	UserRole string `json:"user_role"` // Papel do usuário neste workspace (ex: "admin", "member")
	IsOwner  bool   `json:"is_owner"`  // True se o usuário logado for o dono deste workspace
	// OwnerUID string `json:"owner_uid"` // Opcional: Firebase UID do dono do workspace
}

// ...

type WorkspaceMember struct {
	UserID      string    `json:"user_id"` // Alterado para int64 para corresponder a users.id
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

func CreatePrivateWorkspace(db *sql.DB, ownerUID string) error {
	// Verifica se já existe um workspace privado para esse usuário
	var existingID int64
	err := db.QueryRow(`
	SELECT id FROM workspaces WHERE owner_uid = $1 AND is_public = false
	`, ownerUID).Scan(&existingID)

	if err != nil && err != sql.ErrNoRows {
		return err // Erro de banco real
	}

	if err == nil {
		// Já existe um workspace privado
		return errors.New("private workspace already exists")
	}

	// Inicia uma transação
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		// Rollback se a transação ainda estiver aberta (em caso de erro)
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Cria o workspace privado
	query := `
				INSERT INTO workspaces (name, description, is_public, owner_uid, created_at)
				VALUES ($1, $2, false, $3, NOW())
				RETURNING id, created_at
				`

	name := ownerUID
	description := "Personal workspace"
	// Passo Adicional: Obter o ID numérico do usuário da tabela 'users'
	var userID int64 // Ou o tipo correspondente ao users.id (SERIAL geralmente mapeia para int64 em Go)
	// ownerUID aqui é o firebase_uid (string)
	err = tx.QueryRow("SELECT id FROM users WHERE firebase_uid = $1", ownerUID).Scan(&userID)
	if err != nil {
		log.Printf("Falha ao encontrar ID do usuário na tabela 'users' para firebase_uid %s: %v", ownerUID, err)
		// Este erro é crítico para a lógica de adicionar membro, então a transação deve ser revertida.
		return fmt.Errorf("usuário correspondente ao owner_uid (%s) não encontrado na tabela 'users': %w", ownerUID, err)
	}

	var workspace Workspace
	var createdAt time.Time
	err = tx.QueryRow(
		query,
		name,
		description,
		ownerUID,
	).Scan(&workspace.ID, &createdAt)

	if err != nil {
		return err
	}

	workspace.Name = name
	workspace.Description = description
	workspace.IsPublic = false
	workspace.OwnerUID = ownerUID
	workspace.CreatedAt = createdAt
	workspace.Members = 1

	// Adiciona o dono como admin na tabela workspace_members
	_, err = tx.Exec(`
		INSERT INTO workspace_members (workspace_id, user_id, role, joined_at)
		VALUES ($1, $2, 'admin', NOW())
		`, workspace.ID, userID)

	if err != nil {
		return err
	}

	return nil
}

func ListWorkspaceMembers(db *sql.DB, workspaceID int64) ([]WorkspaceMember, error) {
	query := `
        SELECT u.firebase_uid, u.display_name, u.email, wm.role, wm.joined_at
        FROM workspace_members wm  -- Usando workspace_members consistentemente
        JOIN users u ON wm.user_id = u.id -- JOIN com users.id (inteiro)
        WHERE wm.workspace_id = $1
    `
	rows, err := db.Query(query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar membros do workspace: %w", err)
	}
	defer rows.Close()

	var members []WorkspaceMember
	for rows.Next() {
		var member WorkspaceMember // WorkspaceMember.UserID é string (para firebase_uid)
		err := rows.Scan(
			&member.UserID, // Vai receber u.firebase_uid
			&member.DisplayName,
			&member.Email,
			&member.Role,
			&member.JoinedAt,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	// ... (resto da função)
	return members, nil
}

func GetWorkspaceInfo(db *sql.DB, workspaceID int64) (*Workspace, error) {
	var workspace Workspace
	query := `
		SELECT id, name, description, is_public, owner_uid, created_at
		FROM workspaces
		WHERE id = $1
	`

	err := db.QueryRow(query, workspaceID).Scan(
		&workspace.ID,
		&workspace.Name,
		&workspace.Description,
		&workspace.IsPublic,
		&workspace.OwnerUID,
		&workspace.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("workspace not found")
		}
		return nil, err
	}

	return &workspace, nil
}

func UpdateWorkspace(db *sql.DB, workspaceID int64, name string, description string) error {
	// Validações básicas
	if name == "" {
		return errors.New("workspace name cannot be empty")
	}

	// Query de atualização
	_, err := db.Exec(`
		UPDATE workspaces
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3
	`, name, description, workspaceID)

	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	return nil
}

func DeleteWorkspace(db *sql.DB, workspaceID int64, ownerUID string) error {
	// Verifica se o workspace pertence ao usuário (opcional, mas recomendado para segurança)
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM workspaces
			WHERE id = $1 AND owner_uid = $2
		)
	`, workspaceID, ownerUID).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check workspace ownership: %w", err)
	}

	if !exists {
		return errors.New("workspace not found or user is not the owner")
	}

	// Executa a deleção
	_, err = db.Exec(`
		DELETE FROM workspaces WHERE id = $1
	`, workspaceID)

	if err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	return nil
}

func AddUserToWorkspace(db *sql.DB, workspaceID int64, email string, role string) error {
	if role == "" {
		role = "member"
	}

	var localUserID int64
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&localUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("usuário com email %s não encontrado", email)
		}
		return fmt.Errorf("erro ao buscar ID do usuário: %w", err)
	}

	_, err = db.Exec(`
        INSERT INTO workspace_members (workspace_id, user_id, role, joined_at)
        VALUES ($1, $2, $3, NOW())
    `, workspaceID, localUserID, role) // Usa localUserID (inteiro)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "violates unique constraint") {
			return fmt.Errorf("usuário (email: %s) já é membro do workspace %d", email, workspaceID)
		}
		return fmt.Errorf("falha ao adicionar usuário ao workspace: %w", err)
	}
	return nil
}

func RemoveUserFromWorkspace(db *sql.DB, workspaceID int64, userFirebaseUID string) error {
	var localUserID int64
	err := db.QueryRow("SELECT id FROM users WHERE firebase_uid = $1", userFirebaseUID).Scan(&localUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Usuário não existe, então não pode ser membro. Pode tratar como sucesso ou erro específico.
			return errors.New("usuário não encontrado no sistema para remoção do workspace")
		}
		return fmt.Errorf("erro ao buscar ID do usuário para remoção: %w", err)
	}

	result, err := db.Exec(`
        DELETE FROM workspace_members
        WHERE workspace_id = $1 AND user_id = $2
    `, workspaceID, localUserID) // Usa localUserID (inteiro)
	// ... (resto da função, checagem de rowsAffected)
	if err != nil {
		return fmt.Errorf("falha ao remover usuário do workspace: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("usuário não encontrado no workspace ou já removido")
	}
	return nil
}

func IsWorkspaceMember(db *sql.DB, userFirebaseUID string, workspaceID int64) (bool, error) {
	var localUserID int64
	err := db.QueryRow("SELECT id FROM users WHERE firebase_uid = $1", userFirebaseUID).Scan(&localUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // Usuário não existe no sistema, logo não é membro
		}
		return false, fmt.Errorf("erro ao buscar ID do usuário para checar membresia: %w", err)
	}

	var exists bool
	err = db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM workspace_members
            WHERE user_id = $1 AND workspace_id = $2
        )
    `, localUserID, workspaceID).Scan(&exists) // Usa localUserID (inteiro)

	// ... (resto da função)
	if err != nil {
		return false, fmt.Errorf("falha ao checar se usuário é membro do workspace: %w", err)
	}
	return exists, nil
}
