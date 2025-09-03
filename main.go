package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	paseto "aidanwoods.dev/go-paseto"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
)

var logger *log.Logger

func iniciarLogs() {
	arquivoDeRegistro, err := os.OpenFile("server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Erro ao abrir o aquivo de log!: %v", err)
	}

	logger = log.New(io.MultiWriter(os.Stdout, arquivoDeRegistro), log.Default().Prefix(), log.Lshortfile|log.LstdFlags)
	logger.Printf("\n╒═════════════╡ API BRAIN QUEST ╞═════════════\n│Servidor iniciado em: %v\n│Sistema: %v (%v)\n│Go: %v\n╘═════════════════════════════════════════", time.Now().Format(time.DateTime), runtime.GOOS, runtime.GOARCH, runtime.Version())
}

func OpenConn() (*sql.DB, error) {
	auth := os.Getenv("mariadb")
	db, err := sql.Open("mysql", auth)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	return db, err
}

func main() {
	iniciarLogs()
	testDB, err := OpenConn()
	if err != nil {
		if err.Error() == "pq: database \"brainquest\" does not exist" {
			logger.Fatalln("É necessário que seja inserido no banco as tabelas necessárias.")
		}
		logger.Fatalln(err)
	}
	testDB.Close()

	logger.Println("[i] Verificando chave PASETO...")
	if os.Getenv("paseto_key") == "" {
		logger.Println("[w] Criando chave...")
		key := paseto.NewV4SymmetricKey()
		os.Setenv("paseto_key", key.ExportHex())

		f, _ := os.Open(".env")
		envmap, _ := godotenv.Parse(f)
		envmap["paseto_key"] = key.ExportHex()

		godotenv.Write(envmap, ".env")
	}

	logger.Println("[i] Iniciando rotas...")
	r := http.NewServeMux()

	//Rotas de login
	r.HandleFunc("POST /login/register", registrar)
	r.HandleFunc("POST /login/auth", login)

	//Rotas do usuário
	r.HandleFunc("GET /user/info", userInfo)

	//Rotas das perguntas
	r.HandleFunc("POST /quest/question/{id}", buscarQuestaoId)

	server := http.Server{
		Addr:              os.Getenv("porta"),
		Handler:           r,
		ErrorLog:          logger,
		ReadHeaderTimeout: 10 * time.Second,
	}

	logger.Printf("=> Servidor iniciado com sucesso, endereço: %v,", server.Addr)
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("[e] Erro do servidor: %v", err)
		}
		logger.Println("[w] Sinal recebido para parar o servidor, fechando conexões...")
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
	<-sc

	ctx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("[e] ERRO AO FECHAR SERVIDOR: %v", err)
	}
	logger.Println("[i] Servidor HTTP parado com sucesso.")

}
