# API de Gerenciamento de Usuários, Tarefas e Workspaces

Este é um guia rápido para testar as APIs do sistema usando o Postman.

## Autenticação

### Registro de Usuário
```http
POST /register
Content-Type: application/json

{
    "name": "João Silva",
    "email": "joao@email.com",
    "password": "senha123"
}

Response (200 OK):
{
    "message": "Usuário registrado com sucesso",
    "user": {
        "id": "123",
        "name": "João Silva",
        "email": "joao@email.com"
    }
}
```

### Login
```http
POST /login
Content-Type: application/json

{
    "email": "joao@email.com",
    "password": "senha123"
}

Response (200 OK):
{
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
        "id": "123",
        "name": "João Silva",
        "email": "joao@email.com"
    }
}
```

### Logout
```http
POST /logout
Authorization: Bearer <token>

Response (200 OK):
{
    "message": "Logout realizado com sucesso"
}
```

## Usuários

### Obter Perfil do Usuário
```http
GET /user
Authorization: Bearer <token>

Response (200 OK):
{
    "id": "123",
    "name": "João Silva",
    "email": "joao@email.com"
}
```

### Atualizar Usuário
```http
PUT /user
Authorization: Bearer <token>
Content-Type: application/json

{
    "name": "João Silva Atualizado",
    "email": "joao.novo@email.com"
}

Response (200 OK):
{
    "message": "Usuário atualizado com sucesso",
    "user": {
        "id": "123",
        "name": "João Silva Atualizado",
        "email": "joao.novo@email.com"
    }
}
```

### Deletar Usuário
```http
DELETE /user
Authorization: Bearer <token>

Response (200 OK):
{
    "message": "Usuário deletado com sucesso"
}
```

### Listar Todos os Usuários
```http
GET /users
Authorization: Bearer <token>

Response (200 OK):
{
    "users": [
        {
            "id": "123",
            "name": "João Silva",
            "email": "joao@email.com"
        },
        {
            "id": "124",
            "name": "Maria Santos",
            "email": "maria@email.com"
        }
    ]
}
```

### Obter Usuário Específico
```http
GET /users/{id}
Authorization: Bearer <token>

Response (200 OK):
{
    "id": "123",
    "name": "João Silva",
    "email": "joao@email.com"
}
```

## Workspaces

### Criar Workspace
```http
POST /workspaces
Authorization: Bearer <token>
Content-Type: application/json

{
    "name": "Meu Projeto",
    "description": "Descrição do projeto",
    "is_public": true
}

Response (201 Created):
{
    "id": 1,
    "name": "Meu Projeto",
    "description": "Descrição do projeto",
    "is_public": true,
    "owner_uid": "user123",
    "created_at": "2024-03-20T10:00:00Z",
    "members": 1
}
```

## Tarefas

### Criar Tarefa
```http
POST /workspaces/{workspace_id}/tasks
Authorization: Bearer <token>
Content-Type: application/json

{
    "title": "Implementar API",
    "content": "Desenvolver endpoints REST",
    "priority": "high",
    "status": "pending",
    "expiration": "2024-04-01T23:59:59Z"
}

Response (200 OK):
{
    "id": 1
}
```

### Listar Tarefas
```http
GET /workspaces/{workspace_id}/tasks
Authorization: Bearer <token>

Query Parameters (opcionais):
- status: pending, in_progress, completed
- priority: low, medium, high

Response (200 OK):
[
    {
        "id": 1,
        "title": "Implementar API",
        "content": "Desenvolver endpoints REST",
        "priority": "high",
        "status": "pending",
        "created_by": "João Silva",
        "workspace_id": 1,
        "created_at": "2024-03-20T10:00:00Z",
        "expiration": "2024-04-01T23:59:59Z"
    }
]
```

### Atualizar Tarefa
```http
PUT /tasks/{task_id}
Authorization: Bearer <token>
Content-Type: application/json

{
    "title": "Implementar API REST",
    "content": "Desenvolver endpoints REST com autenticação",
    "priority": "medium",
    "status": "in_progress",
    "expiration": "2024-04-15T23:59:59Z"
}

Response (200 OK):
{
    "message": "Tarefa atualizada com sucesso"
}
```

### Deletar Tarefa
```http
DELETE /tasks/{task_id}
Authorization: Bearer <token>

Response (200 OK):
{
    "message": "Tarefa deletada com sucesso"
}
```

## Observações Importantes

1. Todas as rotas protegidas requerem o token JWT no header `Authorization` no formato `Bearer <token>`
2. O token é obtido após o login bem-sucedido
3. Em caso de erro de autenticação, a resposta será:
```json
{
    "error": "Não autorizado"
}
```

## Códigos de Erro Comuns

- 400: Requisição inválida
- 401: Não autorizado
- 403: Acesso proibido
- 404: Recurso não encontrado
- 500: Erro interno do servidor

## Valores Válidos

### Prioridades de Tarefas
- low
- medium
- high

### Status de Tarefas
- pending
- in_progress
- completed 