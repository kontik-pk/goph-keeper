package database

import (
	"errors"
	"fmt"
)

type ErrDublicateKey struct {
	Key string
}

func (m ErrDublicateKey) Error() string {
	return fmt.Sprintf("ERROR: duplicate key value violates unique constraint %q (SQLSTATE 23505)", m.Key)
}

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrNoSuchUser         = errors.New("no such user")
	ErrInvalidCredentials = errors.New("incorrect password")
	ErrNoData             = errors.New("no data for user")
)
