package internal

import "github.com/golang-jwt/jwt/v4"

type Credentials struct {
	UserName string  `json:"user_name"`
	Login    *string `json:"login,omitempty"`
	Password *string `json:"password,omitempty"`
	Metadata *string `json:"metadata,omitempty"`
}

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type Note struct {
	UserName string  `json:"user_name"`
	Title    *string `json:"title,omitempty"`
	Content  *string `json:"content,omitempty"`
	Metadata *string `json:"metadata,omitempty"`
}

type Card struct {
	UserName string  `json:"user_name"`
	BankName *string `json:"bank_name,omitempty"`
	Number   *string `json:"number,omitempty"`
	CV       *string `json:"cv,omitempty"`
	Password *string `json:"password,omitempty"`
	Metadata *string `json:"metadata,omitempty"`
}

type Params struct {
	StoragePort     string `envconfig:"POSTGRES_PORT"`
	StorageHost     string `envconfig:"POSTGRES_HOST"`
	StorageUser     string `envconfig:"POSTGRES_USER"`
	StoragePassword string `envconfig:"POSTGRES_PASSWORD"`
	StorageDbName   string `envconfig:"POSTGRES_DB"`
	ApplicationPort string `envconfig:"APPLICATION_PORT"`
	ApplicationHost string `envconfig:"APPLICATION_HOST"`
	EncryptionKey   string `envconfig:"KEEPER_ENCRYPTION_KEY"`
}
