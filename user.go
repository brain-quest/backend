package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/paemuri/brdoc"
)

type Register struct {
	Email    string `json:"email"`
	Senha    string `json:"senha"`
	Cpf      string `json:"cpf"`
	Username string `json:"username"`
	Telefone string `json:"telefone"`
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

	if !validarDados(novoUsuario) {
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

func validarDados(r Register) bool {

	if r.Email == "" || r.Cpf == "" || r.Senha == "" || r.Username == "" {
		fmt.Println("1")
		return false
	}

	if !brdoc.IsCPF(r.Cpf) {
		fmt.Println("2")
		return false
	}

	_, err := mail.ParseAddress(r.Email)
	return err == nil
}

func login(w http.ResponseWriter, r *http.Request) {

}

func userInfo(w http.ResponseWriter, r *http.Request) {

}
