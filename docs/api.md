# Documentação da API

Base URL: `https://dev.dataru-ufu.com.br/`  
Formato de resposta: `application/json`  
Autenticação: **PASETO v4.local**, enviado no header `Authorization: Bearer <token>`

---

## Fluxo básico
1. Registrar usuário (`POST /login/register`)
2. Fazer login (`POST /login/auth`) → retorna token
3. Usar token nas outras rotas (`Authorization: Bearer <token>`)

---

## Endpoints

### POST /login/register

#### Descrição
Cria um novo usuário.

#### Requisição
- **Headers:**
  - `Content-Type: application/json`
- **Body (JSON):**
```json
{
  "username": "Fulano",
  "cpf": "12345678900",
  "email": "fulano@ufu.br",
  "senha": "senha123",
  "telefone": "34999999999"
}
```

#### Resposta de Sucesso (201)
```json
{
  "message": "Usuário criado com sucesso"
}
```

#### Possíveis Erros
- **403** → email ou cpf já cadastrados
- **500** → erro interno

---

### POST /login/auth

#### Descrição
Autentica usuário e retorna token.

#### Requisição
- **Headers:**
  - `Content-Type: application/json`
- **Body (JSON):**
```json
{
  "email": "fulano@ufu.br",
  "password": "senha123"
}
```

#### Resposta de Sucesso (200)
```json
{
  "token": "v4.local.ABC...",
  "expiration": 1756339200
}
```

#### Possíveis Erros
- **401** → credenciais inválidas
- **500** → erro interno

---

### GET /user/info

#### Descrição
Retorna informações do usuário autenticado.

#### Requisição
- **Headers:**
  - `Authorization: Bearer <token>`

#### Resposta de Sucesso (200)
```json
{
  "uuid": "5bdb74ca-adb3-4d8a-adc3-f2e420310170",
  "name": "Fulano da Silva",
  "cpf": "123.456.789-00",
  "email": "fulano@ufu.br",
  "telephone": "34999999999",
  "questões_data": {
    "respondidas": 42,
    "acertos": 30,
    "erros": 12,
    "login_streak": 5,
    "last_login": 1756339200
  }
}
```

#### Possíveis Erros
- **400** → token ausente
- **401** → token inválido
- **404** → usuário não encontrado
- **500** → erro interno

---

### GET /quest/question/query/{id}

#### Descrição
Busca questão pelo ID.

#### Requisição
- **Path Params:**
  - `id` → ID numérico da questão
- **Headers:**
  - `Authorization: Bearer <token>`

#### Resposta de Sucesso (200)
```json
{
  "pergunta": "Qual é a capital da França?",
  "alternativa_a": "Paris",
  "alternativa_b": "Roma",
  "alternativa_c": "Berlim",
  "alternativa_d": "Madri",
  "alternativa_e": "Londres"
}
```

#### Possíveis Erros
- **401** → token inválido
- **404** → questão não encontrada
- **500** → erro interno

---

### POST /quest/question/answer/{id}

#### Descrição
Responde uma questão pelo ID e atualiza estatísticas do usuário.

#### Requisição
- **Path Params:**
  - `id` → ID numérico da questão
- **Headers:**
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Body (JSON):**
```json
{
  "alternativa": "A"
}
```

#### Resposta de Sucesso
- **202** → resposta correta
- **204** → resposta incorreta

Exemplo (caso o usuário acerte):
```json
{
    "pergunta": "When was the first offshore deep reservoir in Brazil developed?",
    "resposta": "E"
}
```

#### Possíveis Erros
- **401** → token inválido
- **404** → usuário ou questão não encontrados
- **500** → erro interno
