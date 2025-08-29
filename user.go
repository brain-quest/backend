package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"os"
	"time"

	paseto "aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/paemuri/brdoc"
	"golang.org/x/crypto/bcrypt"
)

type Register struct {
	Email    string  `json:"email"`
	Senha    string  `json:"senha"`
	Cpf      string  `json:"cpf"`
	Username string  `json:"username"`
	Telefone *string `json:"telefone,omitempty"`
}

func registrar(w http.ResponseWriter, r *http.Request) {
	var novoUsuario Register

	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err := d.Decode(&novoUsuario)

	if err != nil || d.More() {
		enviarErrorJson(w, "Estrutura do JSON incorreta.", http.StatusNotAcceptable)
		return
	}

	if !validarDadosRegistrar(novoUsuario) {
		enviarErrorJson(w, "Estrutura do JSON incorreta.", 400)
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(novoUsuario.Senha), bcrypt.DefaultCost)

	ruuid := uuid.New()

	conn, err := OpenConn()
	if err != nil {
		enviarErrorJson(w, "Erro ao conectar ao banco", 504)
		return
	}
	defer conn.Close()

	_, err = conn.Exec("INSERT INTO users (id, email, senha, cpf, nome, telefone) VALUES ($1, $2, $3, $4, $5, $6)", ruuid, novoUsuario.Email, hashedPassword, novoUsuario.Cpf, novoUsuario.Username, novoUsuario.Telefone)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" {
				if pqErr.Constraint == "users_email_key" || pqErr.Constraint == "users_cpf_key" {
					enviarErrorJson(w, "email ou cpf já cadastrado", 403)
					return
				}
				logger.Printf("violação de unicidade: %v", pqErr.Constraint)
				return
			}
		}
		logger.Println(err)
		enviarErrorJson(w, "algo deu errado ao criar usuário", 500)
		return
	}

	enviarRespostaJson(w, "ok", 200)

}

func validarDadosRegistrar(r Register) bool {

	if r.Email == "" || r.Cpf == "" || r.Senha == "" || r.Username == "" {

		return false
	}

	if !brdoc.IsCPF(r.Cpf) {

		return false
	}

	_, err := mail.ParseAddress(r.Email)
	return err == nil
}

type LoginData struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token   string `json:"token"`
	Expires int64  `json:"expiration"`
}

func validarDadosLogin(r LoginData) bool {
	if r.Email == "" || r.Password == "" {
		return false
	}

	_, err := mail.ParseAddress(r.Email)
	return err == nil
}

func login(w http.ResponseWriter, r *http.Request) {
	var dadosLogin LoginData

	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err := d.Decode(&dadosLogin)

	if err != nil || d.More() {
		enviarErrorJson(w, "Estrutura do JSON incorreta.", 401)
		return
	}

	if !validarDadosLogin(dadosLogin) {
		enviarErrorJson(w, "Dados do JSON incorretos.", 401)
		return
	}

	//buscar no pg
	conn, err := OpenConn()
	if err != nil {
		enviarErrorJson(w, "Erro ao conectar ao banco", 504)
		return
	}
	defer conn.Close()

	var senhaSalva, uuidUsuario string

	err = conn.QueryRow("SELECT senha, id FROM users WHERE email = $1", dadosLogin.Email).Scan(&senhaSalva, &uuidUsuario)
	if err == sql.ErrNoRows {
		enviarErrorJson(w, "Usuário ou senha incorretas", 401)
		return
	} else if err != nil {
		enviarErrorJson(w, "Algo deu errado", 500)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(senhaSalva), []byte(dadosLogin.Password))
	if err != nil {
		enviarErrorJson(w, "Usuário ou senha incorretas", 401)
		return
	}

	//devolver token
	token := paseto.NewToken()
	exp := time.Now()

	token.SetIssuedAt(exp)
	token.SetNotBefore(exp)
	token.SetExpiration(exp.Add(2 * time.Hour))

	token.SetString("id", uuidUsuario)

	key, err := paseto.V4SymmetricKeyFromHex(os.Getenv("paseto_key"))
	if err != nil {
		enviarErrorJson(w, "Erro de chave", 500)
		return
	}

	encrypted := token.V4Encrypt(key, nil)

	enviarRespostaJson(w, LoginResponse{Token: encrypted, Expires: exp.Unix()}, 200)
}

type UserData struct {
	UUID      string  `json:"uuid"`
	Name      string  `json:"name"`
	CPF       string  `json:"cpf"`
	Email     string  `json:"email"`
	Telephone *string `json:"telephone,omitempty"`
	Questões  struct {
		Respondidas int   `json:"respondidas"`
		Acertos     int   `json:"acertos"`
		Erros       int   `json:"erros"`
		Dias        int   `json:"login_streak"`
		UltimoLogin int64 `json:"last_login"`
	} `json:"questões_data"`
}

type UserDataFromToken struct {
	User    UserData
	Status  int
	Message string
}

func userInfo(w http.ResponseWriter, r *http.Request) {
	userData := getUserData(r)

	if userData.Status != 200 {
		enviarErrorJson(w, userData.Message, userData.Status)
		return
	}

	enviarRespostaJson(w, userData.User, userData.Status)
}

func getUserData(r *http.Request) UserDataFromToken {
	authToken := r.Header.Get("Authorization")
	if authToken == "" {
		return UserDataFromToken{Message: "Token faltando ou incorreta", Status: 400}
	}

	authToken = authToken[7:]

	key, err := paseto.V4SymmetricKeyFromHex(os.Getenv("paseto_key"))
	if err != nil {
		logger.Println("[e] Erro de chave PASETO:", err)
		return UserDataFromToken{Message: "Erro de chave", Status: 500}
	}

	parseto := paseto.NewParser()
	token, err := parseto.ParseV4Local(key, authToken, nil)
	if err != nil {
		return UserDataFromToken{Message: "Token faltando ou incorreta", Status: 401}
	}

	id, err := token.GetString("id")
	if err != nil {
		return UserDataFromToken{Message: "Token faltando ou incorreta", Status: 400}
	}

	conn, err := OpenConn()
	if err != nil {
		logger.Println("[e] Erro de conexão ao BD:", err)
		return UserDataFromToken{Message: "Não foi possível encontrar o usuário no banco", Status: 504}
	}
	defer conn.Close()

	var userData UserData
	userData.UUID = id

	err = conn.QueryRow(`
    SELECT 
        u.email, u.cpf, u.nome, u.telefone,
        d.quest_feitas, d.alternativas_acertas, d.alternativas_erradas,
        d.dias_logados, EXTRACT(EPOCH FROM d.ultimo_login)::bigint
    FROM users u
    JOIN dados d ON u.id = d.id
    WHERE u.id = $1
`, id).Scan(
		&userData.Email,
		&userData.CPF,
		&userData.Name,
		&userData.Telephone,
		&userData.Questões.Respondidas,
		&userData.Questões.Acertos,
		&userData.Questões.Erros,
		&userData.Questões.Dias,
		&userData.Questões.UltimoLogin,
	)
	if err == sql.ErrNoRows {
		return UserDataFromToken{Message: "O usuário não existe mais", Status: 404}
	} else if err != nil {
		logger.Println("[e] Erro ao buscar dados: ", err)
		return UserDataFromToken{Message: "Algo não deu certo", Status: 500}
	}

	return UserDataFromToken{User: userData, Message: "ok", Status: 200}
}
