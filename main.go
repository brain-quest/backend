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
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

var logger *log.Logger
var ctx = context.Background()
var rdb *redis.Client

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
	logger.Println("[i] Conectando ao MariaDB...")
	if err != nil {
		logger.Fatalln(err)
	}
	testDB.Close()
	logger.Println("[i] MariaDB ok.")
	logger.Println("[i] Conectando ao Redis...")
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       2,
	})
	logger.Println("[i] Redis ok.")

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
	r.HandleFunc("/login/register", registrar)
	r.HandleFunc("/login/auth", login)

	//Rotas do usuário
	r.HandleFunc("/user/info", userInfo)

	//Rotas das perguntas
	r.HandleFunc("/quest/question/query/{id}", buscarQuestaoId)
	//Obtem a pergunta de id {id}
	r.HandleFunc("/quest/question/answer/{id}", responderQuestaoId)
	//Responde a pergunta de {id}

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("healthy."))
	})

	c := cors.New(cors.Options{
		AllowedOrigins:      []string{"*"},
		AllowedMethods:      []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:      []string{"*"},
		AllowCredentials:    true,
		AllowPrivateNetwork: true,
	})

	corsHandler := c.Handler(r)

	server := http.Server{
		Addr:              os.Getenv("porta"),
		Handler:           corsHandler,
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
