package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/paemuri/brdoc"
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

	hasher := sha512.New()
	hasher.Write([]byte(novoUsuario.Senha))
	hashedPassword := hex.EncodeToString(hasher.Sum(nil))

	ruuid := uuid.New()

	conn, err := OpenConn()
	if err != nil {
		enviarErrorJson(w, "deu não", 500)
		return
	}

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
	Token     string  `json:"token"`
	Expires   int64   `json:"expiration"`
	UUID      string  `json:"uuid"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Telephone *string `json:"telephone,omitempty"`
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

	//hashar senha

	//buscar no pg

	//devolver token
}

func userInfo(w http.ResponseWriter, r *http.Request) {

}
