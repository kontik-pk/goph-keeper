package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/kontik-pk/goph-keeper/internal"
	"github.com/kontik-pk/goph-keeper/internal/database"
	"io"
	"net/http"
	"strings"
	"time"
)

func parseInputUser(r io.ReadCloser) (*internal.User, error) {
	var userFromRequest *internal.User
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, fmt.Errorf("error while reading request body: %w", err)
	}
	if err := json.Unmarshal(buf.Bytes(), &userFromRequest); err != nil {
		return nil, fmt.Errorf("error while unmarshalling request body: %w", err)
	}
	if userFromRequest.Login == "" || userFromRequest.Password == "" {
		return nil, fmt.Errorf("login or password is empty")
	}
	return userFromRequest, nil
}

func extractJwtToken(cookies string) (*jwt.Token, error) {
	splitted := strings.Split(cookies, " ")
	if len(splitted) != 2 {
		return nil, ErrNoToken
	}

	tknStr := splitted[1]
	claims := &internal.Claims{}
	tkn, err := jwt.ParseWithClaims(tknStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	return tkn, err
}

func createToken(userName string, expirationTime time.Time) (string, error) {
	claims := &internal.Claims{
		Username: userName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func parseUserError(userName string, err error) (string, int) {
	if errors.Is(err, database.ErrNoSuchUser) {
		return fmt.Sprintf("no such user %q", userName), http.StatusUnauthorized
	}
	if errors.Is(err, database.ErrInvalidCredentials) {
		return fmt.Sprintf("provided password is wrong for user %q", userName), http.StatusUnauthorized
	}
	if errors.Is(err, database.ErrUserAlreadyExists) {
		return fmt.Sprintf("login %q is already taken", userName), http.StatusConflict
	}
	if errors.Is(err, database.ErrNoData) {
		return fmt.Sprintf("no data for user %q", userName), http.StatusNoContent
	}
	if errors.Is(err, jwt.ErrSignatureInvalid) ||
		errors.Is(err, jwt.ErrTokenExpired) ||
		errors.Is(err, ErrTokenIsEmpty) ||
		errors.Is(err, ErrNoToken) {
		return fmt.Sprintf("token problem for user %q: %s", userName, err.Error()), http.StatusUnauthorized
	}
	return fmt.Sprintf("user %q request error : %s", userName, err.Error()), http.StatusInternalServerError
}
