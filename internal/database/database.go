package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/kontik-pk/goph-keeper/internal"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type db struct {
	conn          *sql.DB
	encriptionKey string
	dataCipher    cipher.Block
}

func New(params internal.Params) (*db, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		params.StorageHost, params.StoragePort, params.StorageUser, params.StoragePassword, params.StorageDbName)
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error while trying to open DB connection: %w", err)
	}
	c, err := aes.NewCipher([]byte(params.EncryptionKey))
	if err != nil {
		return nil, fmt.Errorf("error while creation cipher with key: %w", err)
	}
	pg := db{
		conn:          conn,
		encriptionKey: params.EncryptionKey,
		dataCipher:    c,
	}

	if err = pg.conn.Ping(); err != nil {
		return nil, fmt.Errorf("error while trying to ping DB: %w", err)
	}
	return &pg, nil
}

// SaveNote is a method for saving provided notes (note title, content and probably metadata)
// for authorized user in goph-keeper storage.
func (d *db) SaveNote(ctx context.Context, noteRequest internal.Note) error {
	encryptedContent, err := d.encryptAES(*noteRequest.Content)
	if err != nil {
		return fmt.Errorf("error encrypting your classified text: %w", err)
	}
	saveNotesQuery := "insert into notes (user_name, title, content, metadata) values ($1, $2, $3, $4)"
	if _, err = d.conn.ExecContext(ctx, saveNotesQuery, noteRequest.UserName, noteRequest.Title, encryptedContent, noteRequest.Metadata); err != nil {
		return fmt.Errorf("error while saving note for user %q: %w", noteRequest.UserName, err)
	}
	return nil
}

// GetNotes is a method for getting notes (note title, content and probably metadata) for
// provided authorized user from goph-keeper storage.
func (d *db) GetNotes(ctx context.Context, noteRequest internal.Note) ([]internal.Note, error) {
	args := []any{noteRequest.UserName}
	getNoteQuery := "select user_name, title, content, metadata from notes where user_name = $1"
	if noteRequest.Title != nil {
		args = append(args, *noteRequest.Title)
		getNoteQuery += fmt.Sprintf(" and title = $%d", len(args))
	}
	rows, err := d.conn.QueryContext(ctx, getNoteQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error while getting notes for user %q: %w", noteRequest.UserName, err)
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()

	var notes []internal.Note
	for rows.Next() {
		var userName, title, content string
		var metadata sql.NullString
		if err = rows.Scan(&userName, &title, &content, &metadata); err != nil {
			return nil, fmt.Errorf("error while scanning rows after get user notes query: %w", err)
		}
		decryptedContent, err := d.decryptAES(content)
		if err != nil {
			return nil, fmt.Errorf("error while decrypting password: %w", err)
		}
		res := internal.Note{
			UserName: userName,
			Title:    &title,
			Content:  &decryptedContent,
		}
		if metadata.Valid {
			res.Metadata = &metadata.String
		}
		notes = append(notes, res)
	}
	if len(notes) == 0 {
		return nil, ErrNoData
	}
	return notes, nil
}

// DeleteNotes is a method for deleting notes for provided user. Title is optional parameter.
func (d *db) DeleteNotes(ctx context.Context, noteRequest internal.Note) error {
	args := []any{noteRequest.UserName}
	deleteNotesQuery := "delete from notes where user_name= $1"
	if noteRequest.Title != nil {
		args = append(args, *noteRequest.Title)
		deleteNotesQuery += " and title = $2"
	}
	deleteCredsQuery := "delete from notes where user_name= $1 and title=$2"
	if _, err := d.conn.ExecContext(ctx, deleteCredsQuery, args...); err != nil {
		return fmt.Errorf("error while deleting note for user %q: %w", noteRequest.UserName, err)
	}
	return nil
}

// UpdateNote is a method for updating note content for authorized user in goph-keeper storage.
func (d *db) UpdateNote(ctx context.Context, noteRequest internal.Note) error {
	encryptedContent, err := d.encryptAES(*noteRequest.Content)
	if err != nil {
		return fmt.Errorf("error encrypting note content: %w", err)
	}
	updateNoteQuery := "update notes set content = $1, metadata = $2 where user_name = $3 and title = $4"
	if _, err = d.conn.ExecContext(ctx, updateNoteQuery, encryptedContent, noteRequest.Metadata, noteRequest.UserName, *noteRequest.Title); err != nil {
		return fmt.Errorf("error while updating note %q for user %q: %w", *noteRequest.Title, noteRequest.UserName, err)
	}
	return nil
}

// SaveCredentials is a method for saving provided credentials (pair of login/password and probably metadata)
// for authorized user in goph-keeper storage.
func (d *db) SaveCredentials(ctx context.Context, credentialsRequest internal.Credentials) error {
	encryptedPassword, err := d.encryptAES(*credentialsRequest.Password)
	if err != nil {
		return fmt.Errorf("error encrypting your classified text: %w", err)
	}
	saveCredsQuery := "insert into credentials (user_name, login, password, metadata) values ($1, $2, $3, $4)"
	if _, err = d.conn.ExecContext(ctx, saveCredsQuery, credentialsRequest.UserName, *credentialsRequest.Login, encryptedPassword, credentialsRequest.Metadata); err != nil {
		return fmt.Errorf("error while saving credentials for user %q: %w", credentialsRequest.UserName, err)
	}
	return nil
}

// GetCredentials is a method for getting credentials (pair of login/password and probably metadata) for
// provided authorized user from goph-keeper storage.
func (d *db) GetCredentials(ctx context.Context, credentialsRequest internal.Credentials) ([]internal.Credentials, error) {
	args := []any{credentialsRequest.UserName}
	getCredsQuery := "select user_name, login, password, metadata from credentials where user_name = $1"
	if credentialsRequest.Login != nil {
		args = append(args, *credentialsRequest.Login)
		getCredsQuery += fmt.Sprintf(" and login = $%d", len(args))
	}
	rows, err := d.conn.QueryContext(ctx, getCredsQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error while getting credentials for user %q: %w", credentialsRequest.UserName, err)
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()

	var creds []internal.Credentials
	for rows.Next() {
		var userName, login, password string
		var metadata sql.NullString
		if err = rows.Scan(&userName, &login, &password, &metadata); err != nil {
			return nil, fmt.Errorf("error while scanning rows after get user credentials query: %w", err)
		}
		decryptedPassword, err := d.decryptAES(password)
		if err != nil {
			return nil, fmt.Errorf("error while decrypting password: %w", err)
		}
		res := internal.Credentials{
			UserName: userName,
			Login:    &login,
			Password: &decryptedPassword,
		}
		if metadata.Valid {
			res.Metadata = &metadata.String
		}
		creds = append(creds, res)
	}
	if len(creds) == 0 {
		return nil, ErrNoData
	}
	return creds, nil
}

// DeleteCredentials is a method for deleting all credentials for provided user. Login is optional parameter.
func (d *db) DeleteCredentials(ctx context.Context, credentialsRequest internal.Credentials) error {
	args := []any{credentialsRequest.UserName}
	deleteCredsQuery := "delete from credentials where user_name= $1"
	if credentialsRequest.Login != nil {
		args = append(args, *credentialsRequest.Login)
		deleteCredsQuery += " and login=$2"
	}
	if _, err := d.conn.ExecContext(ctx, deleteCredsQuery, args...); err != nil {
		return fmt.Errorf("error while deleting credentials for user %q: %w", credentialsRequest.UserName, err)
	}
	return nil
}

// UpdateCredentials is a method for updating credentials (pair of login/password and probably metadata)
// for authorized user in goph-keeper storage.
func (d *db) UpdateCredentials(ctx context.Context, credentialsRequest internal.Credentials) error {
	encryptedPassword, err := d.encryptAES(*credentialsRequest.Password)
	if err != nil {
		return fmt.Errorf("error encrypting your classified text: %w", err)
	}
	updateCredsQuery := "update credentials set password = $1, metadata = $2 where user_name = $3 and login = $4"
	if _, err = d.conn.ExecContext(ctx, updateCredsQuery, encryptedPassword, credentialsRequest.Metadata, credentialsRequest.UserName, credentialsRequest.Login); err != nil {
		return fmt.Errorf("error while updating credentials for user %q: %w", credentialsRequest.UserName, err)
	}
	return nil
}

// SaveCard is a method for saving provided bank card (bank name, card number, cv, password probably metadata)
// for authorized user in goph-keeper storage.
func (d *db) SaveCard(ctx context.Context, cardRequest internal.Card) error {
	encryptedPassword, err := d.encryptAES(*cardRequest.Password)
	if err != nil {
		return fmt.Errorf("error encrypting card password: %w", err)
	}
	encryptedCV, err := d.encryptAES(*cardRequest.CV)
	if err != nil {
		return fmt.Errorf("error encrypting card password: %w", err)
	}
	saveCardQuery := "insert into cards (user_name, bank_name, number, cv, password, metadata) values ($1, $2, $3, $4, $5, $6)"
	if _, err = d.conn.ExecContext(ctx, saveCardQuery, cardRequest.UserName, *cardRequest.BankName, *cardRequest.Number, encryptedCV, encryptedPassword, cardRequest.Metadata); err != nil {
		return fmt.Errorf("error while saving card data for user %q: %w", cardRequest.UserName, err)
	}
	return nil
}

// GetCard is a method for getting user's bank cards (bank name, number, cv, password and probably metadata) for
// provided authorized user from goph-keeper storage.
func (d *db) GetCard(ctx context.Context, cardRequest internal.Card) ([]internal.Card, error) {
	args := []any{cardRequest.UserName}
	getCardsQuery := "select user_name, bank_name, number, cv, password, metadata from cards where user_name = $1"
	if cardRequest.BankName != nil {
		args = append(args, *cardRequest.BankName)
		getCardsQuery += fmt.Sprintf(" and bank_name = $%d", len(args))
	}
	if cardRequest.Number != nil {
		args = append(args, *cardRequest.Number)
		getCardsQuery += fmt.Sprintf(" and number = $%d", len(args))
	}
	rows, err := d.conn.QueryContext(ctx, getCardsQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error while getting cards for user %q: %w", cardRequest.UserName, err)
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()

	var cards []internal.Card
	for rows.Next() {
		var userName, bankName, number, cv, password string
		var metadata sql.NullString
		if err = rows.Scan(&userName, &bankName, &number, &cv, &password, &metadata); err != nil {
			return nil, fmt.Errorf("error while scanning rows after get user notes query: %w", err)
		}
		decryptedPassword, err := d.decryptAES(password)
		if err != nil {
			return nil, fmt.Errorf("error while decrypting password: %w", err)
		}
		decryptedCV, err := d.decryptAES(cv)
		if err != nil {
			return nil, fmt.Errorf("error while decrypting password: %w", err)
		}
		res := internal.Card{
			UserName: userName,
			BankName: &bankName,
			Number:   &number,
			CV:       &decryptedCV,
			Password: &decryptedPassword,
		}
		if metadata.Valid {
			res.Metadata = &metadata.String
		}
		cards = append(cards, res)
	}
	if len(cards) == 0 {
		return nil, ErrNoData
	}
	return cards, nil
}

// DeleteCards is a method for deleting bank cards for provided user. Bank name and number are optional parameters.
func (d *db) DeleteCards(ctx context.Context, cardRequest internal.Card) error {
	args := []any{cardRequest.UserName}
	deleteNotesQuery := "delete from cards where user_name= $1"
	if cardRequest.Number != nil {
		args = append(args, *cardRequest.Number)
		deleteNotesQuery += fmt.Sprintf(" and number = $%d", len(args))
	}
	if cardRequest.BankName != nil {
		args = append(args, *cardRequest.BankName)
		deleteNotesQuery += fmt.Sprintf(" and bank_name = $%d", len(args))
	}
	if _, err := d.conn.ExecContext(ctx, deleteNotesQuery, args...); err != nil {
		return fmt.Errorf("error while deleting cards for user %q: %w", cardRequest.UserName, err)
	}
	return nil
}

// Login is a method for login user in goph-keeper system with provided login and password.
// User logins are stored in the goph-keeper database as bcrypt hashes.
// Provided password is hashed and the result is compared with the content from database.
func (d *db) Login(ctx context.Context, login string, password string) error {
	getRegisteredUser := `select login, password from registered_users where login = $1`

	var loginFromDB, passwordFromDB string
	if err := d.conn.QueryRowContext(ctx, getRegisteredUser, login).Scan(&loginFromDB, &passwordFromDB); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoSuchUser
		}
		return fmt.Errorf("error while executing search query: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordFromDB), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

// Register is a method for register new user in goph-keeper storage with provided credentials.
func (d *db) Register(ctx context.Context, login string, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("this password is not allowed: %w", err)
	}
	registerUser := `insert into registered_users values ($1, $2)`
	if _, err = d.conn.ExecContext(ctx, registerUser, login, hash); err != nil {
		dublicateKeyErr := ErrDublicateKey{Key: "registered_users_pkey"}
		if err.Error() == dublicateKeyErr.Error() {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("error while executing register user query: %w", err)
	}
	return nil
}

// Close is a method for closing database connection.
func (d *db) Close() error {
	return d.conn.Close()
}

func (d *db) encryptAES(plaintext string) (string, error) {
	cfb := cipher.NewCFBEncrypter(d.dataCipher, []byte(d.encriptionKey)[:aes.BlockSize])
	cipherText := make([]byte, len(plaintext))
	cfb.XORKeyStream(cipherText, []byte(plaintext))
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func (d *db) decryptAES(ct string) (string, error) {
	cipherText, err := base64.StdEncoding.DecodeString(ct)
	if err != nil {
		return "", err
	}

	cfb := cipher.NewCFBDecrypter(d.dataCipher, []byte(d.encriptionKey)[:aes.BlockSize])
	plainText := make([]byte, len(cipherText))
	cfb.XORKeyStream(plainText, cipherText)

	return string(plainText), nil
}
