# API de Gerenciamento de Usuários, Tarefas e Workspaces

Este é um guia rápido para testar as APIs do sistema usando o Postman ou similar.

## Autenticação

### 1. Registro de Novo Usuário
Cria um novo usuário no Firebase Authentication e no banco de dados PostgreSQL. Também cria um workspace privado padrão para o usuário.
```http
POST /auth/register
Content-Type: application/json

{
    "displayName": "João Silva",
    "email": "joao@email.com",
    "password": "senha123"
}
```
**Response (201 Created):**
```json
{
    "message": "User created successfully and ready to sign in",
    "uid": "FIREBASE_UID_DO_NOVO_USUARIO",
    "customToken": "CUSTOM_FIREBASE_TOKEN_PARA_LOGIN_IMEDIATO"
}
```

### 2. Finalizar Login com Token Firebase
Processa um ID Token do Firebase (obtido após login com email/senha ou social no cliente/frontend) para verificar o usuário e sincronizá-lo com o banco de dados local.
```http
POST /auth/finalize-login
Content-Type: application/json

{
    "idToken": "ID_TOKEN_OBTIDO_DO_FIREBASE_CLIENT_SDK"
}
```
**Response (200 OK):**
```json
{
    "message": "Login finalizado e usuário sincronizado com sucesso.",
    "firebaseUid": "FIREBASE_UID_DO_USUARIO"
}
```

### 3. Logout do Usuário
Revoga os tokens de atualização do Firebase para o usuário autenticado.
```http
POST /auth/logout
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Response (200 OK):**
```json
{
    "message": "Logout efetuado com sucesso"
}
```

## Usuário Autenticado

### 1. Obter Informações do Perfil do Usuário Logado
Retorna informações do usuário atualmente autenticado.
```http
GET /user/info
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Response (200 OK):**
```json
{
    "id": 1, 
    "firebase_uid": "FIREBASE_UID_DO_USUARIO",
    "email": "joao@email.com",
    "display_name": "João Silva"
}
```

### 2. Atualizar Informações do Usuário Logado
Atualiza o nome de exibição do usuário autenticado.
```http
PUT /user/update
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "display_name": "João Silva Atualizado"
}
```
**Response (200 OK):**
```json
{
    "message": "User updated successfully"
}
```

### 3. Listar Meus Workspaces
Lista todos os workspaces dos quais o usuário autenticado é membro.
```http
GET /user/my-workspaces/list
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Response (200 OK):**
```json
[
    {
        "id": 1,
        "name": "Meu Workspace Privado",
        "user_role": "admin",
        "is_owner": true
    },
    {
        "id": 5,
        "name": "Projeto Equipe Alpha",
        "user_role": "member",
        "is_owner": false
    }
]
```

## Usuários (Operações Gerais)

### 1. Listar Todos os Usuários do Sistema
Retorna uma lista de todos os usuários registrados.
```http
GET /users/list
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Response (200 OK):**
```json
[
    {
        "id": 1, 
        "firebase_uid": "FIREBASE_UID_JOAO",
        "email": "joao@email.com",
        "display_name": "João Silva"
    },
    {
        "id": 2,
        "firebase_uid": "FIREBASE_UID_MARIA",
        "email": "maria@email.com",
        "display_name": "Maria Santos"
    }
]
```

### 2. Obter Informações de um Usuário Específico
Retorna informações de um usuário específico com base no seu ID numérico do PostgreSQL.
```http
GET /users/info/{id}
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Exemplo de Path:** `/users/info/1`
**Response (200 OK):**
```json
{
    "id": 1,
    "firebase_uid": "FIREBASE_UID_JOAO",
    "email": "joao@email.com",
    "display_name": "João Silva"
}
```

## Workspaces

### 1. Criar Novo Workspace
Cria um novo workspace. O usuário autenticado se torna o dono e admin.
```http
POST /workspace/create
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "name": "Projeto Phoenix",
    "description": "Workspace para o desenvolvimento do Projeto Phoenix.",
    "is_public": false
}
```
**Response (201 Created):**
```json
{
    "id": 2, 
    "name": "Projeto Phoenix",
    "description": "Workspace para o desenvolvimento do Projeto Phoenix.",
    "is_public": false,
    "owner_uid": "FIREBASE_UID_DO_CRIADOR",
    "created_at": "2025-05-29T18:00:00Z", 
    "members": 1 
}
```

### 2. Obter Informações de um Workspace
Busca detalhes de um workspace específico pelo seu ID. Requer que o usuário seja membro ou que o workspace seja público.
```http
GET /workspace/info/{workspace_id}
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Exemplo de Path:** `/workspace/info/2`
**Response (200 OK):**
```json
{
    "id": 2,
    "name": "Projeto Phoenix",
    "description": "Workspace para o desenvolvimento do Projeto Phoenix.",
    "is_public": false,
    "owner_uid": "FIREBASE_UID_DO_CRIADOR",
    "created_at": "2025-05-29T18:00:00Z",
    "members": 3 
}
```

### 3. Atualizar um Workspace
Atualiza o nome e/ou descrição de um workspace. Somente o dono pode atualizar.
```http
PUT /workspace/update/{workspace_id}
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "name": "Projeto Phoenix (Atualizado)",
    "description": "Descrição atualizada do Projeto Phoenix."
}
```
**Exemplo de Path:** `/workspace/update/2`
**Response (200 OK):**
```json
{
    "message": "Workspace updated successfully"
}
```

### 4. Deletar um Workspace
Deleta um workspace e todos os seus dados associados. Somente o dono pode deletar.
```http
DELETE /workspace/delete/{workspace_id}
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Exemplo de Path:** `/workspace/delete/2`
**Response (204 No Content)**

### 5. Listar Membros de um Workspace
Lista todos os membros de um workspace específico. Requer que o usuário seja membro do workspace.
```http
GET /workspace/{workspace_id}/members/list
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Exemplo de Path:** `/workspace/2/members/list`
**Response (200 OK):**
```json
[
    {
        "user_id": "FIREBASE_UID_DO_MEMBRO1", 
        "display_name": "Membro Um",
        "email": "membro1@email.com",
        "role": "admin",
        "joined_at": "2025-05-29T18:00:00Z"
    },
    {
        "user_id": "FIREBASE_UID_DO_MEMBRO2",
        "display_name": "Membro Dois",
        "email": "membro2@email.com",
        "role": "member",
        "joined_at": "2025-05-29T18:05:00Z"
    }
]
```

### 6. Adicionar Usuário a um Workspace
Adiciona um usuário a um workspace com um papel específico. Somente o dono pode adicionar membros.
```http
POST /workspace/{workspace_id}/members/add
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "userFirebaseUid": "FIREBASE_UID_DO_USUARIO_A_ADICIONAR",
    "role": "member" // ou "admin"
}
```
**Exemplo de Path:** `/workspace/2/members/add`
**Response (201 Created):**
```json
{
    "message": "User added to workspace successfully"
}
```

### 7. Remover Usuário de um Workspace
Remove um usuário de um workspace. O dono pode remover outros membros (exceto ele mesmo). Um usuário também pode remover a si próprio.
```http
DELETE /workspace/{workspace_id}/members/remove
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "userFirebaseUid": "FIREBASE_UID_DO_USUARIO_A_REMOVER"
}
```
**Exemplo de Path:** `/workspace/2/members/remove`
**Response (204 No Content)**

## Tarefas

### 1. Criar Nova Tarefa em um Workspace
Cria uma nova tarefa associada a um workspace. Requer que o usuário seja membro do workspace.
```http
POST /workspace/{workspace_id}/task/create
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "title": "Implementar Autenticação de Dois Fatores",
    "description": "Detalhes sobre a implementação de 2FA usando TOTP.",
    "status": "pending",
    "priority": "high",
    "expiration_date": "2025-08-15T23:59:59Z"
}
```
**Exemplo de Path:** `/workspace/2/task/create`
**Response (201 Created):**
```json
{
    "id": "ID_GERADO_PELO_FIRESTORE", 
    "title": "Implementar Autenticação de Dois Fatores",
    "description": "Detalhes sobre a implementação de 2FA usando TOTP.",
    "status": "pending",
    "priority": "high",
    "expirationDate": "2025-08-15T23:59:59Z",
    "creatorFirebaseUid": "FIREBASE_UID_DO_CRIADOR",
    "createdAt": "2025-05-29T19:00:00Z", 
    "lastUpdatedAt": "2025-05-29T19:00:00Z" 
}
```

### 2. Listar Tarefas de um Workspace
Lista todas as tarefas de um workspace. Requer que o usuário seja membro do workspace.
```http
GET /workspace/{workspace_id}/task/list
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Exemplo de Path:** `/workspace/2/task/list`
**Response (200 OK):**
```json
[
    {
        "id": "FIRESTORE_DOC_ID_DA_TAREFA_1", 
        "title": "Implementar Autenticação de Dois Fatores",
        "description": "Detalhes sobre a implementação de 2FA usando TOTP.",
        "status": "pending",
        "priority": "high",
        "expirationDate": "2025-08-15T23:59:59Z",
        "creatorFirebaseUid": "FIREBASE_UID_DO_CRIADOR",
        "createdAt": "2025-05-29T19:00:00Z",
        "lastUpdatedAt": "2025-05-29T19:00:00Z"
    }
]
```

### 3. Obter Detalhes de uma Tarefa Específica
Busca os detalhes de uma tarefa específica pelo seu ID de documento do Firestore.
```http
GET /workspace/{workspace_id}/task/info/{task_doc_id}
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Exemplo de Path:** `/workspace/2/task/info/FIRESTORE_DOC_ID_DA_TAREFA`
**Response (200 OK):**
```json
{
    "id": "FIRESTORE_DOC_ID_DA_TAREFA",
    "title": "Implementar Autenticação de Dois Fatores",
    "description": "Detalhes sobre a implementação de 2FA usando TOTP.",
    "status": "pending",
    "priority": "high",
    "expirationDate": "2025-08-15T23:59:59Z",
    "creatorFirebaseUid": "FIREBASE_UID_DO_CRIADOR",
    "createdAt": "2025-05-29T19:00:00Z",
    "lastUpdatedAt": "2025-05-29T19:00:00Z"
}
```

### 4. Atualizar uma Tarefa
Atualiza os detalhes de uma tarefa existente.
```http
PUT /workspace/{workspace_id}/task/update/{task_doc_id}
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "title": "Implementar 2FA (Revisado e Testado)",
    "status": "in_progress",
    "priority": "medium"
}
```
**Exemplo de Path:** `/workspace/2/task/update/FIRESTORE_DOC_ID_DA_TAREFA`
**Response (200 OK):**
```json
{
    "message": "Task updated successfully"
}
```

### 5. Deletar uma Tarefa
Deleta uma tarefa do sistema.
```http
DELETE /workspace/{workspace_id}/task/delete/{task_doc_id}
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Exemplo de Path:** `/workspace/2/task/delete/FIRESTORE_DOC_ID_DA_TAREFA`
**Response (204 No Content)**

## Funcionalidades de Inteligência Artificial

### 1. Revisão de Código
Envia um trecho de código para a IA e recebe uma revisão detalhada.
```http
POST /workspace/{workspace_id}/ai/code-review
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "code": "func exemplo(a int, b int) int {\n  return a + b\n}\n\nfunc main() {\n  soma := exemplo(5, 10)\n  fmt.Println(soma)\n}",
    "language": "Go"
}
```
**Response (200 OK):**
```json
{
    "review": "Um code review detalhado fornecido pela IA, com sugestões de melhoria, performance, segurança, etc."
}
```

### 2. Resumo de Texto
Envia um texto para a IA e recebe um resumo conciso.
```http
POST /workspace/{workspace_id}/ai/summarize-text
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "text": "O Projeto Integrador é um sistema abrangente que visa otimizar a gestão de tarefas e a colaboração em workspaces..."
}
```
**Response (200 OK):**
```json
{
    "summary": "Um resumo conciso do texto fornecido pela IA, identificando os pontos chave."
}
```

### 3. Geração de Ideias para Mapa Mental
Envia um texto para a IA e recebe sugestões de tópicos e subtópicos para um mapa mental.
```http
POST /workspace/{workspace_id}/ai/mindmap-ideas
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "text": "Estou planejando as férias de verão. Os principais pontos a considerar são: destino, orçamento, atividades..."
}
```
**Response (200 OK):**
```json
{
    "mind_map_ideas": "Uma lista hierárquica de tópicos e subtópicos para um mapa mental, fornecida pela IA."
}
```

### 4. Assistente de Tarefas do Workspace
Envia uma mensagem do usuário para a API de IA, que usa o contexto do workspace para fornecer sugestões.
```http
POST /workspace/{workspace_id}/ai/task-assistant
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "user_message": "Quais são as tarefas mais urgentes que estão pendentes neste workspace?"
}
```
**Response (200 OK):**
```json
{
    "suggestions": [
        "Sugestão 1 da IA baseada no contexto do workspace e na mensagem do usuário.",
        "Sugestão 2 da IA..."
    ]
}
```

## Observações Importantes

1. Todas as rotas protegidas requerem o `ID Token` do Firebase (obtido no cliente após login) no header `Authorization` no formato `Bearer <ID_TOKEN_DO_FIREBASE>`.
2. O endpoint `/auth/finalize-login` é usado para processar este `ID Token` no backend e sincronizar o usuário com o banco de dados local.
3. Em caso de erro de autenticação/autorização, a resposta geralmente será um status HTTP 401 (Não Autorizado) ou 403 (Proibido).

## Códigos de Erro Comuns

- **200 OK:** Requisição bem-sucedida.
- **201 Created:** Recurso criado com sucesso.
- **204 No Content:** Requisição bem-sucedida, sem conteúdo para retornar.
- **400 Bad Request:** Requisição inválida (ex: JSON malformado, parâmetros faltando).
- **401 Unauthorized:** Autenticação falhou ou token ausente/inválido.
- **403 Forbidden:** Autenticado, mas sem permissão para acessar/modificar o recurso.
- **404 Not Found:** Recurso não encontrado.
- **409 Conflict:** Conflito, por exemplo, tentar registrar um email que já existe.
- **500 Internal Server Error:** Erro inesperado no servidor.

## Valores Válidos

### Prioridades de Tarefas (`priority`)
- `low`
- `medium`
- `high`

### Status de Tarefas (`status`)
- `pending`
- `in_progress`
- `completed`

### Papéis (Roles) de Membros em Workspaces (`role`)
- `admin`
- `member`