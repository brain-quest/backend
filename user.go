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
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
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
	if r.Method != http.MethodOptions && r.Method != http.MethodPost {
		w.WriteHeader(406)
		return
	}
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(novoUsuario.Senha), bcrypt.DefaultCost)
	if err != nil {
		enviarErrorJson(w, "Falha ao criar hash do password", 500)
		return
	}

	ruuid := uuid.New().String()

	conn, err := OpenConn()
	if err != nil {
		enviarErrorJson(w, "Erro ao conectar ao banco", 504)
		return
	}
	defer conn.Close()

	_, err = conn.Exec("INSERT INTO users (id, email, senha, cpf, nome, telefone) VALUES (?, ?, ?, ?, ?, ?)", ruuid, novoUsuario.Email, hashedPassword, novoUsuario.Cpf, novoUsuario.Username, novoUsuario.Telefone)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			if mysqlErr.Number == 1062 { // ER_DUP_ENTRY
				enviarErrorJson(w, "email ou cpf já cadastrado", http.StatusForbidden)
				return
			}
		}

		logger.Println("[e] Erro ao inserir usuário:", err)
		enviarErrorJson(w, "algo deu errado ao criar usuário", http.StatusInternalServerError)
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
	if r.Method != http.MethodOptions && r.Method != http.MethodPost {
		w.WriteHeader(406)
		return
	}
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

	err = conn.QueryRow("SELECT senha, id FROM users WHERE email = ?", dadosLogin.Email).Scan(&senhaSalva, &uuidUsuario)
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

	//atualiza o ultimo_login, e verifica o streak
	if _, err := conn.Exec(`UPDATE dados
    SET 
      ultimo_login = CURDATE(),
      dias_logados = CASE
        WHEN DATEDIFF(CURDATE(), ultimo_login) = 1 THEN dias_logados + 1
        WHEN DATEDIFF(CURDATE(), ultimo_login) = 0 THEN dias_logados
        ELSE 1
      END
    WHERE id = ?;`,
		uuidUsuario,
	); err != nil {
		// se der errado, ignorar e continua o login
		logger.Printf("[w] falha ao atualizar ultimo_login de %v: %v\n", uuidUsuario, err)
	}

	//devolver token
	token := paseto.NewToken()
	//exp := time.Now()

	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	token.SetExpiration(time.Now().Add(36 * time.Hour))

	token.SetString("id", uuidUsuario)

	key, err := paseto.V4SymmetricKeyFromHex(os.Getenv("paseto_key"))
	if err != nil {
		enviarErrorJson(w, "Erro de chave", 500)
		return
	}

	encrypted := token.V4Encrypt(key, nil)
	exp, _ := token.GetExpiration()

	enviarRespostaJson(w, LoginResponse{Token: encrypted, Expires: exp.Unix()}, 200)
}

type UserData struct {
	UUID      string  `json:"uuid"`
	Name      string  `json:"name"`
	CPF       string  `json:"cpf"`
	Email     string  `json:"email"`
	Telephone *string `json:"telephone,omitempty"`
	Questões  struct {
		Respondidas       int      `json:"respondidas"`
		Acertos           int      `json:"acertos"`
		Erros             int      `json:"erros"`
		Dias              int      `json:"login_streak"`
		UltimoLogin       int64    `json:"last_login"`
		QuestõesFeitas    []string `json:"feitas,omitempty"`
		QuestõesAcertadas []string `json:"acertadas,omitempty"`
		Quizzes           []string `json:"quizzes,omitempty"`
	} `json:"questões_data"`
}

type UserDataFromToken struct {
	User    UserData
	Status  int
	Message string
}

func userInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodOptions && r.Method != http.MethodGet {
		w.WriteHeader(406)
		return
	}
	userData := getUserData(r)

	if userData.Status != 200 {
		enviarErrorJson(w, userData.Message, userData.Status)
		return
	}

	questoesFeitas, err := listarQuestoesFeitas(userData.User.UUID)
	if err != nil {
		logger.Printf("[w] Não foi possível achar as questões feitas por %v: %v\n", userData.User.UUID, err)
	}
	questoesAcertadas, err := listarQuestoesAcertadas(userData.User.UUID)
	if err != nil {
		logger.Printf("[w] Não foi possível achar as questões acertadas por %v: %v\n", userData.User.UUID, err)
	}
	quizzesFeitos, err := listarQuizzesFeitos(userData.User.UUID)
	if err != nil {
		logger.Printf("[w] Não foi possível achar os quizzes feitos por %v: %v\n", userData.User.UUID, err)
	}

	userData.User.Questões.QuestõesAcertadas = questoesAcertadas
	userData.User.Questões.QuestõesFeitas = questoesFeitas
	userData.User.Questões.Quizzes = quizzesFeitos

	enviarRespostaJson(w, userData.User, userData.Status)
}

type UserUUID struct {
	Status  int
	Message string
	UUID    string
}

func getUserUUID(r *http.Request) UserUUID {
	authToken := r.Header.Get("Authorization")
	if authToken == "" {
		return UserUUID{Message: "Token faltando ou incorreta", Status: 400}
	}

	authToken = authToken[7:]

	key, err := paseto.V4SymmetricKeyFromHex(os.Getenv("paseto_key"))
	if err != nil {
		logger.Println("[e] Erro de chave PASETO:", err)
		return UserUUID{Message: "Erro de chave", Status: 500}
	}

	parseto := paseto.NewParser()
	token, err := parseto.ParseV4Local(key, authToken, nil)
	if err != nil {
		return UserUUID{Message: "Token faltando ou incorreta", Status: 401}
	}

	id, err := token.GetString("id")
	if err != nil {
		return UserUUID{Message: "Token faltando ou incorreta", Status: 400}
	}

	return UserUUID{Status: 200, Message: "Usuário OK", UUID: id}
}

func getUserData(r *http.Request) UserDataFromToken {
	id := getUserUUID(r)
	if id.Status != 200 {
		return UserDataFromToken{Message: id.Message, Status: id.Status}
	}

	var userData UserData
	userData.UUID = id.UUID

	conn, err := OpenConn()
	if err != nil {
		logger.Println("[e] Erro de conexão ao BD:", err)
		return UserDataFromToken{Message: "Não foi possível encontrar o usuário no banco", Status: 504}
	}
	defer conn.Close()

	err = conn.QueryRow(`
    SELECT 
        u.email, u.cpf, u.nome, u.telefone,
        d.quest_feitas, d.alternativas_acertas, d.alternativas_erradas,
        d.dias_logados, UNIX_TIMESTAMP(d.ultimo_login)
    FROM users u
    JOIN dados d ON u.id = d.id
    WHERE u.id = ?
`, id.UUID).Scan(
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
