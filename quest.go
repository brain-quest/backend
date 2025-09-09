package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type Pergunta struct {
	Pergunta     string `json:"pergunta"`
	AlternativaA string `json:"alternativa_a,omitempty"`
	AlternativaB string `json:"alternativa_b,omitempty"`
	AlternativaC string `json:"alternativa_c,omitempty"`
	AlternativaD string `json:"alternativa_d,omitempty"`
	AlternativaE string `json:"alternativa_e,omitempty"`
	Resposta     string `json:"resposta,omitempty"`
}

func buscarQuestaoId(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodOptions && r.Method != http.MethodGet {
		w.WriteHeader(406)
		return
	}
	questionID := r.PathValue("id")
	qid, err := strconv.Atoi(questionID)
	if questionID == "" || err != nil {
		enviarErrorJson(w, "ID da pergunta vazio", 400)
		return
	}
	uid := getUserUUID(r)
	if uid.Status != 200 {
		enviarErrorJson(w, "Token de autorização inválida", 403)
		return
	}

	conn, err := OpenConn()
	if err != nil {
		enviarErrorJson(w, "Erro ao conectar ao banco", 504)
		return
	}
	defer conn.Close()

	var exists bool
	err = conn.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", uid.UUID).Scan(&exists)
	if err != nil {
		enviarErrorJson(w, "Usuário não existe", 404)
		logger.Println("[e] Erro ao buscar usuário:", err)
		return
	}
	if !exists {
		enviarErrorJson(w, "Usuário inexistente", 404)
		return
	}
	var pergunta Pergunta
	err = conn.QueryRow("SELECT pergunta, alternativa_a, alternativa_b, alternativa_c, alternativa_d, alternativa_e FROM questoes WHERE id = ?", qid).Scan(&pergunta.Pergunta, &pergunta.AlternativaA, &pergunta.AlternativaB, &pergunta.AlternativaC, &pergunta.AlternativaD, &pergunta.AlternativaE)
	if err == sql.ErrNoRows {
		enviarErrorJson(w, "ID da pergunta incorreto", 401)
		return
	} else if err != nil {
		logger.Println("[e] Erro ao buscar pergunta:", err)
		enviarErrorJson(w, "Algo deu errado", 500)
		return
	}

	enviarRespostaJson(w, pergunta, 200)
}

type RespostaQuiz struct {
	Alternativa string `json:"alternativa"`
}

func responderQuestaoId(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodOptions && r.Method != http.MethodPost {
		w.WriteHeader(406)
		return
	}
	questionID := r.PathValue("id")
	qid, err := strconv.Atoi(questionID)
	if questionID == "" || err != nil {
		enviarErrorJson(w, "ID da pergunta vazio", 400)
		return
	}
	uid := getUserUUID(r)
	if uid.Status != 200 {
		enviarErrorJson(w, "Token de autorização inválida", 403)
		return
	}

	var dadosResposta RespostaQuiz

	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err = d.Decode(&dadosResposta)

	if err != nil || d.More() {
		enviarErrorJson(w, "Estrutura do JSON incorreta.", 401)
		return
	}

	verificarIfDone, err := usuarioJaFez(uid.UUID, qid)
	if err != nil {
		logger.Printf("[w] Não foi possível verificar se %v já fez a questão %v: %v\n", uid.UUID, qid, err)
	}
	if verificarIfDone {
		enviarErrorJson(w, "Usuário já respondeu essa pergunta", 409)
		return
	}

	conn, err := OpenConn()
	if err != nil {
		enviarErrorJson(w, "Erro ao conectar ao banco", 504)
		return
	}
	defer conn.Close()

	var exists bool
	err = conn.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", uid.UUID).Scan(&exists)
	if err != nil {
		enviarErrorJson(w, "Usuário não existe", 404)
		logger.Println("[e] Erro ao buscar usuário:", err)
		return
	}
	if !exists {
		enviarErrorJson(w, "Usuário inexistente", 404)
		return
	}
	var pergunta Pergunta
	err = conn.QueryRow("SELECT pergunta, correta FROM questoes WHERE id = ?", questionID).Scan(&pergunta.Pergunta, &pergunta.Resposta)
	if err == sql.ErrNoRows {
		enviarErrorJson(w, "ID da pergunta incorreto", 401)
		return
	} else if err != nil {
		logger.Println("[e] Erro ao buscar pergunta:", err)
		enviarErrorJson(w, "Algo deu errado", 500)
		return
	}

	acertou := dadosResposta.Alternativa == pergunta.Resposta

	sqlUpdate := `
    UPDATE dados
    SET 
      quest_feitas = quest_feitas + 1,
      alternativas_acertas = alternativas_acertas + ?,
      alternativas_erradas = alternativas_erradas + ?
    WHERE id = ?;
`

	acertos := 0
	erros := 0
	if acertou {
		acertos = 1
	} else {
		erros = 1
	}

	if _, err := conn.Exec(sqlUpdate, acertos, erros, uid.UUID); err != nil {
		logger.Printf("[w] falha ao atualizar os dados de %v: %v\n", uid, err)
	}

	if err := registrarResposta(uid.UUID, qid, acertou); err != nil {
		logger.Printf("[w] falha ao atualizar Redis de %v: %v\n", uid.UUID, err)
	}

	if !acertou {
		enviarRespostaJson(w, pergunta, 204)
		return
	}

	enviarRespostaJson(w, pergunta, 202)
}

func usuarioJaFez(userID string, questaoID int) (bool, error) {
	key := fmt.Sprintf("user:%s:feitas", userID)
	return rdb.SIsMember(ctx, key, questaoID).Result()
}

func usuarioAcertou(userID string, questaoID int) (bool, error) {
	key := fmt.Sprintf("user:%s:acertos", userID)
	return rdb.SIsMember(ctx, key, questaoID).Result()
}

func listarQuestoesFeitas(userID string) ([]string, error) {
	key := fmt.Sprintf("user:%s:feitas", userID)
	return rdb.SMembers(ctx, key).Result()
}

func listarQuestoesAcertadas(userID string) ([]string, error) {
	key := fmt.Sprintf("user:%s:acertos", userID)
	return rdb.SMembers(ctx, key).Result()
}

func registrarResposta(userID string, questaoID int, acertou bool) error {
	keyFeitas := fmt.Sprintf("user:%s:feitas", userID)
	keyAcertos := fmt.Sprintf("user:%s:acertos", userID)

	//add a feitas
	if err := rdb.SAdd(ctx, keyFeitas, questaoID).Err(); err != nil {
		return err
	}

	//add a acertadas
	if acertou {
		if err := rdb.SAdd(ctx, keyAcertos, questaoID).Err(); err != nil {
			return err
		}
	}

	return nil
}
