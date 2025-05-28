-- Tabela de usuários
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    firebase_uid VARCHAR(128) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de workspaces
CREATE TABLE workspaces (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT false,
    owner_uid VARCHAR(128) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_uid) REFERENCES users(firebase_uid)
);

-- Tabela de membros do workspace
CREATE TABLE workspace_members (
    workspace_id INTEGER REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'member')),
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, user_id)
);

CREATE TABLE tarefas (
    id SERIAL PRIMARY KEY,                          -- ID interno único no PostgreSQL
    firestore_doc_id VARCHAR(128) UNIQUE NOT NULL,  -- ID do documento correspondente no Firestore (essencial para o link)
    workspace_id INTEGER NOT NULL,                  -- ID do workspace ao qual a tarefa pertence
    criado_por INTEGER NOT NULL,                    -- ID do usuário (da tabela 'users') que criou a tarefa
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (criado_por) REFERENCES users(id) ON DELETE RESTRICT -- Ou ON DELETE SET NULL/CASCADE dependendo da sua lógica de negócio
);

-- Índice para a nova coluna
CREATE INDEX idx_tarefas_firestore_doc_id ON tarefas(firestore_doc_id);
-- Índices existentes que ainda são relevantes (se você não deu DROP TABLE)
-- Se você deu DROP TABLE, precisa recriá-los para a nova estrutura:
CREATE INDEX idx_tarefas_workspace ON tarefas(workspace_id);
CREATE INDEX idx_tarefas_criado_por ON tarefas(criado_por);

-- Índices para melhor performance
CREATE INDEX idx_users_firebase_uid ON users(firebase_uid);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_workspaces_owner ON workspaces(owner_uid);
CREATE INDEX idx_tarefas_workspace ON tarefas(workspace_id);
CREATE INDEX idx_tarefas_criado_por ON tarefas(criado_por);
CREATE INDEX idx_workspace_members_user ON workspace_members(user_id);
CREATE INDEX idx_workspace_members_workspace ON workspace_members(workspace_id);

-- Função para atualizar o updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers para atualizar updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workspaces_updated_at
    BEFORE UPDATE ON workspaces
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tarefas_updated_at
    BEFORE UPDATE ON tarefas
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column(); 