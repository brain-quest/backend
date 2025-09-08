package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type MsgErro struct {
	Erro     bool   `json:"erro"`
	Mensagem string `json:"mensagem"`
}

func enviarErrorJson(w http.ResponseWriter, msg string, status int) {
	s := MsgErro{
		Erro:     true,
		Mensagem: msg,
	}

	enviarRespostaJson(w, s, status)
}

func enviarRespostaJson(w http.ResponseWriter, resposta any, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resposta)
}

func enviarRespostaString(w http.ResponseWriter, resposta string, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	fmt.Fprintln(w, resposta)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
