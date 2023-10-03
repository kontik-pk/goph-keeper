package handler

import "errors"

var (
	ErrTokenIsEmpty = errors.New("token is empty")
	ErrNoToken      = errors.New("no token")
)
