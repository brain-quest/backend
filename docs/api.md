# Endpoints da API

## /login/register

| **Meteodo** |   **Caminho**   | **Body** |
|:-----------:|:---------------:|:--------:|
|     POST    | /login/register |   JSON   |

Exemplo de JSON:

```json
{
    "email":"example@ufu.br",
    "senha":"12345678",
    "cpf":"000.000.000-00",
    "username":"um nome",
    "telefone":"00123456789"
}
```

Se não houver telefone, a chave `telefone` não deve estar presente.

### Resposta

| **Status** |                **Motivo/Mensagem**                |
|:----------:|:-------------------------------------------------:|
|     200    |        Dados ok, registro feito com sucesso       |
|     400    |         Teste do CPF/Email retornaram erro        |
|     403    | Já existe um usuário com o email ou CPF informado |
|     500    |             Erro interno da aplicação             |

## /login/auth

| **Meteodo** |   **Caminho**   | **Body** |
|:-----------:|:---------------:|:--------:|
|     POST    |   /login/auth   |   JSON   |

Exemplo de JSON:

```json
{
    "email":"example@ufu.br",
    "password":"12345678",
}
```

### Resposta de Autenticação

| **Status** |        **Motivo/Mensagem**        |
|:----------:|:---------------------------------:|
|     200    | Dados ok, login feito com sucesso |
|     401    |      Email ou senha incorreta     |
|     500    |     Erro interno da aplicação     |

Se o status for 200, o seguinte JSON será retornado:

```json
{
    "token": "v4.local.TokenPaseto....",
    "expiration": 1756429734
}
```

## /user/info

| **Meteodo** |   **Caminho**   | **Header** |
|:-----------:|:---------------:|:--------:|
|     GET     |   /user/info   |   Authorization   |

É necessário o header com a token de autorização, que deve ser recebida após realizar o login no endpoint acima.

| **Status** |             **Motivo/Mensagem**            |
|:----------:|:------------------------------------------:|
|     200    |   Dados ok, JSON com os dados do usuário   |
|     400    |         Token faltando ou incorreta        |
|     401    |               Token inválida.              |
|     404    | Token válida, mas usuário não existe mais. |
|     500    |          Erro interno da aplicação         |
|     506    |       Erro ao buscar usuário no banco      |

JSON de resposta, caso 200:

```json
{
    "uuid": "5bdb74ca-adb3-4d8a-adc3-f2e420310170",
    "name": "um nome",
    "cpf": "000.000.000-00",
    "email": "example@ufu.br",
    "questões_data": {
        "respondidas": 0,
        "acertos": 0,
        "erros": 0,
        "login_streak": 0,
        "last_login": 1756339200
    }
}
```

## /quest/question/{id}

(POST)
header necessário: authorization, com a token
body: json com a letra da pergunta
{id}: id da pergunta

+= Resposta
202: acertou;
204: errou;
400: faltou a token;
401: token invalida.
