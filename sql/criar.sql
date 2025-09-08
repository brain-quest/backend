DROP TABLE IF EXISTS dados;
DROP TABLE IF EXISTS questoes;
DROP TABLE IF EXISTS users;

CREATE TABLE users (
    id CHAR(36) PRIMARY KEY NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    senha TEXT NOT NULL,
    cpf VARCHAR(20) NOT NULL UNIQUE,
    nome VARCHAR(255) NOT NULL,
    telefone VARCHAR(20)
);

CREATE TABLE questoes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    pergunta TEXT NOT NULL,
    alternativa_a TEXT NOT NULL,
    alternativa_b TEXT NOT NULL,
    alternativa_c TEXT NOT NULL,
    alternativa_d TEXT NOT NULL,
    alternativa_e TEXT NOT NULL,
    correta CHAR(1) NOT NULL
);

CREATE TABLE dados (
    id CHAR(36) NOT NULL,
    quest_feitas INT NOT NULL DEFAULT 0,
    alternativas_acertas INT NOT NULL DEFAULT 0,
    alternativas_erradas INT NOT NULL DEFAULT 0,
    dias_logados INT NOT NULL DEFAULT 0,
    ultimo_login DATE NOT NULL DEFAULT (CURRENT_DATE),
    CONSTRAINT fk_dados_users FOREIGN KEY (id) REFERENCES users(id)
);

DELIMITER $$

CREATE TRIGGER after_user_insert
AFTER INSERT ON users
FOR EACH ROW
BEGIN
    INSERT INTO dados (id, ultimo_login)
    VALUES (NEW.id, CURRENT_DATE);
END$$

DELIMITER ;
