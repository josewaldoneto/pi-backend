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

### 2. Finalizar Login com Token Firebase (Pós-Login no Cliente)
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

## Usuário Autenticado (Operações do Próprio Usuário)

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

### 3. Deletar Conta do Usuário Logado
Deleta o usuário autenticado. (Nota: Handler correspondente `DeleteUserHandler` precisa ser implementado).
```http
DELETE /user/delete
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Response (200 OK ou 204 No Content):**
```json
{
    "message": "Usuário deletado com sucesso" 
}
```

### 4. Listar Meus Workspaces
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

## Usuários (Operações Gerais - Geralmente para Admins ou Consultas Específicas)

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

### 2. Obter Informações de um Usuário Específico (por ID numérico do PG)
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
**Response (204 No Content ou 200 OK com o workspace atualizado):**
(Se 204, não há corpo de resposta. Se 200 OK, pode retornar o objeto do workspace atualizado.)

### 4. Deletar um Workspace
Deleta um workspace e todos os seus dados associados (membros, stubs de tarefas no PG, documentos de tarefas e o workspace no Firestore). Somente o dono pode deletar.
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
Adiciona um usuário a um workspace com um papel específico. Somente o dono (ou, futuramente, um admin do workspace) pode adicionar membros.
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
**Response (201 Created ou 200 OK):**
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

## Tarefas (dentro de um Workspace)
(Modelo Híbrido: Stub no PostgreSQL, Detalhes e Tempo Real no Firestore)

### 1. Criar Nova Tarefa em um Workspace
Cria uma nova tarefa associada a um workspace. Requer que o usuário seja membro do workspace.
```http
POST /workspace/{workspace_id}/task/create
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "title": "Implementar Autenticação de Dois Fatores",
    "description": "Detalhes sobre a implementação de 2FA usando TOTP.",
    "status": "pending", // "pending", "in_progress", "completed"
    "priority": "high",  // "low", "medium", "high"
    "expiration_date": "2025-08-15T23:59:59Z", // Opcional, formato ISO 8601
    "attachment": { // Opcional
        "filename": "especificacao_2fa.pdf",
        "url": "[https://storage.example.com/path/to/especificacao_2fa.pdf](https://storage.example.com/path/to/especificacao_2fa.pdf)", // URL direta do arquivo
        "filetype": "application/pdf"
    }
}
```
**Exemplo de Path:** `/workspace/2/task/create`
**Response (201 Created):**
(Retorna os detalhes da tarefa como criados no Firestore, incluindo o ID do documento Firestore, similar à resposta de "Obter Detalhes de uma Tarefa")
```json
{
    "id": "ID_GERADO_PELO_FIRESTORE", 
    "title": "Implementar Autenticação de Dois Fatores",
    "description": "Detalhes sobre a implementação de 2FA usando TOTP.",
    "status": "pending",
    "priority": "high",
    "expirationDate": "2025-08-15T23:59:59Z",
    "attachment": {
        "filename": "especificacao_2fa.pdf",
        "url": "[https://storage.example.com/path/to/especificacao_2fa.pdf](https://storage.example.com/path/to/especificacao_2fa.pdf)",
        "filetype": "application/pdf"
    },
    "creatorFirebaseUid": "FIREBASE_UID_DO_CRIADOR",
    "createdAt": "2025-05-29T19:00:00Z", 
    "lastUpdatedAt": "2025-05-29T19:00:00Z" 
}
```

### 2. Listar Tarefas de um Workspace
Lista todas as tarefas de um workspace (buscando detalhes do Firestore). Requer que o usuário seja membro do workspace.
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
        "attachment": { /* ... */ },
        "creatorFirebaseUid": "FIREBASE_UID_DO_CRIADOR",
        "createdAt": "2025-05-29T19:00:00Z",
        "lastUpdatedAt": "2025-05-29T19:00:00Z"
    }
    
]
```

### 3. Obter Detalhes de uma Tarefa Específica
Busca os detalhes de uma tarefa específica pelo seu ID de documento do Firestore, dentro de um workspace. Requer que o usuário seja membro.
```http
GET /workspace/{workspace_id}/task/info/{task_doc_id}
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Exemplo de Path:** `/workspace/2/task/info/FIRESTORE_DOC_ID_DA_TAREFA`
**Response (200 OK):**
(Retorna o objeto da tarefa como no exemplo de Listar Tarefas, mas para uma única tarefa)
```json
{
    "id": "FIRESTORE_DOC_ID_DA_TAREFA",
    "title": "Implementar Autenticação de Dois Fatores",
    "description": "Detalhes sobre a implementação de 2FA usando TOTP.",
    "status": "pending",
    "priority": "high",
    "expirationDate": "2025-08-15T23:59:59Z",
    "attachment": { /* ... */ },
    "creatorFirebaseUid": "FIREBASE_UID_DO_CRIADOR",
    "createdAt": "2025-05-29T19:00:00Z",
    "lastUpdatedAt": "2025-05-29T19:00:00Z"
}
```

### 4. Atualizar uma Tarefa
Atualiza os detalhes de uma tarefa existente no Firestore (e o `updated_at` do stub no PG). Requer que o usuário seja membro (e, idealmente, tenha permissão de edição na tarefa).
```http
PUT /workspace/{workspace_id}/task/update/{task_doc_id}
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json

{
    "title": "Implementar 2FA (Revisado e Testado)",
    "status": "in_progress",
    "priority": "medium"
    // Envie apenas os campos que deseja atualizar
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
Deleta uma tarefa do Firestore e seu stub correspondente do PostgreSQL. Requer que o usuário seja membro (e, idealmente, tenha permissão de deleção).
```http
DELETE /workspace/{workspace_id}/task/delete/{task_doc_id}
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
```
**Exemplo de Path:** `/workspace/2/task/delete/FIRESTORE_DOC_ID_DA_TAREFA`
**Response (204 No Content)**

## Funcionalidades de Inteligência Artificial (IA)
(Requer Autenticação: `Authorization: Bearer <ID_TOKEN_DO_FIREBASE>`)

### 1. Revisão de Código (Code Review)
Envia um trecho de código para a IA e recebe uma revisão detalhada.
```http
POST /workspace/{workspace_id}/ai/code-review
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json
```
**Exemplo de Path:** `/workspace/1/ai/code-review`

**Corpo da Requisição:**
```json
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
**Possível Erro (400 Bad Request):**
```json
{
    "error": "O campo 'code' é obrigatório para code review"
}
```

### 2. Resumo de Documento/Texto
Envia um texto para a IA e recebe um resumo conciso.
```http
POST /workspace/{workspace_id}/ai/summarize-text
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json
```
**Exemplo de Path:** `/workspace/1/ai/summarize-text`

**Corpo da Requisição:**
```json
{
    "text": "O Projeto Integrador é um sistema abrangente que visa otimizar a gestão de tarefas e a colaboração em workspaces. Ele utiliza tecnologias modernas como Go para o backend, PostgreSQL para o banco de dados relacional e Cloud Firestore para funcionalidades em tempo real e armazenamento de detalhes de tarefas, garantindo escalabilidade e segurança. As funcionalidades incluem criação e gerenciamento de tarefas, administração de workspaces e um sistema robusto de autenticação de usuários via Firebase Authentication. O objetivo final é prover uma plataforma intuitiva e eficiente para equipes melhorarem sua produtividade e comunicação em projetos diversos, com o auxílio de um assistente de IA integrado."
}
```
**Response (200 OK):**
```json
{
    "summary": "Um resumo conciso do texto fornecido pela IA, identificando os pontos chave."
}
```
**Possível Erro (400 Bad Request):**
```json
{
    "error": "O campo 'text' é obrigatório para resumo"
}
```

### 3. Geração de Ideias para Mapa Mental
Envia um texto para a IA e recebe sugestões de tópicos e subtópicos para um mapa mental.
```http
POST /workspace/{workspace_id}/ai/mindmap-ideas
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json
```
**Exemplo de Path:** `/workspace/1/ai/mindmap-ideas`

**Corpo da Requisição:**
```json
{
    "text": "Estou planejando as férias de verão. Os principais pontos a considerar são: destino (praia ou montanha), orçamento total disponível, atividades para crianças, opções de transporte (carro ou avião) e duração da viagem (uma ou duas semanas)."
}
```
**Response (200 OK):**
```json
{
    "mind_map_ideas": "Uma lista hierárquica de tópicos e subtópicos para um mapa mental, fornecida pela IA (ex: - Destino\n  -- Praia\n  -- Montanha\n- Orçamento\n  -- ...)."
}
```
**Possível Erro (400 Bad Request):**
```json
{
    "error": "O campo 'text' é obrigatório para mapa mental"
}
```

### 4. Assistente de Tarefas do Workspace
Envia uma mensagem do usuário para a API de IA, que usa o contexto do workspace (buscado pelo backend Go) para fornecer sugestões ou assistência relacionada às tarefas.
```http
POST /workspace/{workspace_id}/ai/task-assistant
Authorization: Bearer <ID_TOKEN_DO_FIREBASE>
Content-Type: application/json
```
**Exemplo de Path:** `/workspace/8/ai/task-assistant` (onde `8` é o `workspace_id`)

**Corpo da Requisição:**
(O `workspace_id` já está no path, o corpo só precisa da mensagem do usuário)
```json
{
    "user_message": "Quais são as tarefas mais urgentes que estão pendentes neste workspace?"
}
```
Ou:
```json
{
    "user_message": "Preciso de ideias para novas tarefas para o projeto 'Lançamento do App Mobile'."
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
**Possível Erro (400 Bad Request):**
```json
{
    "error": "Mensagem do usuário é obrigatória"
}
```

## Observações Importantes

1.  Todas as rotas protegidas (a maioria delas) requerem o `ID Token` do Firebase (obtido no cliente após login) no header `Authorization` no formato `Bearer <ID_TOKEN_DO_FIREBASE>`.
2.  O endpoint `/auth/finalize-login` é usado para processar este `ID Token` no backend e sincronizar o usuário com o banco de dados local.
3.  Em caso de erro de autenticação/autorização, a resposta geralmente será um status HTTP 401 (Não Autorizado) ou 403 (Proibido).

## Códigos de Erro Comuns

- **200 OK:** Requisição bem-sucedida.
- **201 Created:** Recurso criado com sucesso.
- **204 No Content:** Requisição bem-sucedida, sem conteúdo para retornar (comum em DELETE ou PUTs que não retornam o objeto).
- **400 Bad Request:** Requisição inválida (ex: JSON malformado, parâmetros faltando, validação falhou).
- **401 Unauthorized:** Autenticação falhou ou token ausente/inválido.
- **403 Forbidden:** Autenticado, mas sem permissão para acessar/modificar o recurso.
- **404 Not Found:** Recurso não encontrado.
- **409 Conflict:** Conflito, por exemplo, tentar registrar um email que já existe ou criar um recurso que já existe com um identificador único.
- **500 Internal Server Error:** Erro inesperado no servidor.

## Valores Válidos (Exemplos)

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
```