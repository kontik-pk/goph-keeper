package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kontik-pk/goph-keeper/internal"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"time"
)

var jwtKey = []byte("my_secret_key")

type handler struct {
	db      internal.Storage
	log     *zap.SugaredLogger
	cookies map[string]string
}

func New(db internal.Storage, log *zap.SugaredLogger) *handler {
	return &handler{
		db:      db,
		log:     log,
		cookies: make(map[string]string),
	}
}

// Login is a method for login in goph-keeper system.
// The body of the HTTP request must contain `login` and `password`.
// For example: curl -X POST http://127.0.0.1:8080/auth/login `{"login": "user_login", "password": "user_password"}`
func (h *handler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body and get user's login/password
	user, err := parseInputUser(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// check password for user
	if err = h.db.Login(ctx, user.Login, user.Password); err != nil {
		message, status := parseUserError(user.Login, err)
		http.Error(w, message, status)
		return
	}
	// create jwt token for user, add Authorization header and remember cookie
	expirationTime := time.Now().Add(time.Hour)
	token, err := createToken(user.Login, expirationTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("error while create token for user: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Authorization", fmt.Sprintf("Bearer %s", token))
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   token,
		Expires: expirationTime,
	})
	w.WriteHeader(http.StatusOK)
	h.cookies[user.Login] = fmt.Sprintf("Bearer %s", token)
	h.log.Infof("user %q was successfully logined", user.Login)
}

// Register is a method for register user in goph-keeper system with provided credentials.
// The body of the HTTP request must contain `login` and `password`.
// For example: curl -X POST http://127.0.0.1:8080/auth/register --data `{"login": "user_login", "password": "user_password"}`
func (h *handler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body and get user's login/password
	user, err := parseInputUser(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// register user in goph-keeper system
	if err = h.db.Register(ctx, user.Login, user.Password); err != nil {
		message, status := parseUserError(user.Login, err)
		http.Error(w, message, status)
		return
	}

	// create jwt token for user, add Authorization header and remember cookie
	expirationTime := time.Now().Add(time.Hour)
	token, err := createToken(user.Login, expirationTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("error while create token for user: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Authorization", fmt.Sprintf("Bearer %s", token))
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   token,
		Expires: expirationTime,
	})
	w.WriteHeader(http.StatusOK)
	h.cookies[user.Login] = fmt.Sprintf("Bearer %s", token)
	h.log.Infof("user %q was successfully registered", user.Login)
}

// GetUserCredentials is a method for getting credentials (pair of login/password and probably metadata)
// for provided authorized user. The body of the HTTP request must contain user's name.
// For example: curl -X POST http://127.0.0.1:8080/get/credentials --data `{"user_name": "some_name"}`
func (h *handler) GetUserCredentials(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body to get user's name
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var userCredentialsRequest internal.Credentials
	if err = json.Unmarshal(body, &userCredentialsRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// get user credentials from goph-keeper storage
	creds, err := h.db.GetCredentials(ctx, userCredentialsRequest)
	if err != nil {
		message, status := parseUserError(userCredentialsRequest.UserName, err)
		http.Error(w, message, status)
		return
	}

	// response
	credsResponse, err := json.Marshal(creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err = io.WriteString(w, string(credsResponse)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// SaveUserCredentials is a method for saving provided credentials (pair of login/password and probably metadata)
// for authorized user. The body of the HTTP request must contain user's name, login, password. Metadata is optional.
// For example:
// curl -X POST http://127.0.0.1:8080/save/credentials --data `{"user_name": "some_name", "login": "some_login", "password": "strong_password", "metadata": "some optional data"}`
func (h *handler) SaveUserCredentials(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var requestCredentials internal.Credentials
	if err := json.Unmarshal(buf.Bytes(), &requestCredentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if requestCredentials.Login == nil || requestCredentials.Password == nil {
		http.Error(w, "login and password should not be empty", http.StatusBadRequest)
		return
	}

	// save credentials for user in goph-keeper storage
	if err := h.db.SaveCredentials(ctx, requestCredentials); err != nil {
		message, status := parseUserError(requestCredentials.UserName, err)
		http.Error(w, message, status)
		return
	}

	// response
	if _, err := io.WriteString(w, fmt.Sprintf("saved credentials for user %q", requestCredentials.UserName)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// DeleteUserCredentials is a method for deleting credentials for provided user.
// CredentialsRequest body must contain user's name, login is optional.
// For example:
// curl -X POST http://127.0.0.1:8080/delete/credentials --data `{"user_name": "some_name", "login": "some_login"}`
func (h *handler) DeleteUserCredentials(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body to get user's name
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var userCredentialsRequest internal.Credentials
	if err = json.Unmarshal(body, &userCredentialsRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// delete credentials from goph-keeper storage
	if err := h.db.DeleteCredentials(ctx, userCredentialsRequest); err != nil {
		message, status := parseUserError(userCredentialsRequest.UserName, err)
		http.Error(w, message, status)
		return
	}
	response := fmt.Sprintf("credentials for user %q was successfully deleted", userCredentialsRequest.UserName)
	if userCredentialsRequest.Login != nil {
		response = fmt.Sprintf("credentials for user %q with login %q was successfully deleted", userCredentialsRequest.UserName, *userCredentialsRequest.Login)
	}

	if _, err = io.WriteString(w, response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

}

// UpdateUserCredentials is a method for updating password and metadata
// for authorized user with provided login. CredentialsRequest body must contain user's name, login, password. Metadata is optional.
// For example:
// curl -X POST http://127.0.0.1:8080/update/credentials --data `{"user_name": "some_name", "login": "some_login", "password": "new_password", "metadata": "some optional data"}`
func (h *handler) UpdateUserCredentials(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var requestCredentials internal.Credentials
	if err := json.Unmarshal(buf.Bytes(), &requestCredentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if requestCredentials.Login == nil || requestCredentials.Password == nil {
		http.Error(w, "login and password should not be empty", http.StatusBadRequest)
		return
	}

	// update credentials for user in goph-keeper storage
	if err := h.db.UpdateCredentials(ctx, requestCredentials); err != nil {
		message, status := parseUserError(requestCredentials.UserName, err)
		http.Error(w, message, status)
		return
	}

	// response
	if _, err := io.WriteString(w, fmt.Sprintf("updated credentials for user %q", requestCredentials.UserName)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// SaveUserNote is a method for saving provided note (note title, content and probably metadata)
// for authorized user. Request body must contain user's name, title, note content. Metadata is optional.
// For example:
// curl -X POST http://127.0.0.1:8080/save/note --data `{"user_name": "some_name", "title": "note_title", "content": "shopping list", "metadata": "some optional data"}`
func (h *handler) SaveUserNote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var requestNote internal.Note
	if err := json.Unmarshal(buf.Bytes(), &requestNote); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if requestNote.Title == nil || requestNote.UserName == "" {
		http.Error(w, "user name and note title should not be empty", http.StatusBadRequest)
		return
	}

	// save note for user in goph-keeper storage
	if err := h.db.SaveNote(ctx, requestNote); err != nil {
		message, status := parseUserError(requestNote.UserName, err)
		http.Error(w, message, status)
		return
	}

	// response
	if _, err := io.WriteString(w, fmt.Sprintf("saved note for user %q", requestNote.UserName)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetUserNote is a method for getting user's note (title, content and probably metadata)
// for provided authorized user. CredentialsRequest body must contain user's name.
// For example: curl -X POST http://127.0.0.1:8080/get/note --data `{"user_name": "some_name", "title": "some_title"}`
func (h *handler) GetUserNote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body to get user's name
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var userNotesRequest internal.Note
	if err = json.Unmarshal(body, &userNotesRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// get user note from goph-keeper storage
	creds, err := h.db.GetNotes(ctx, userNotesRequest)
	if err != nil {
		message, status := parseUserError(userNotesRequest.UserName, err)
		http.Error(w, message, status)
		return
	}

	// response
	notesResponse, err := json.Marshal(creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err = io.WriteString(w, string(notesResponse)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// DeleteUserNotes is a method for deleting notes for provided user.
// CredentialsRequest body must contain user's name, title is optional.
// For example:
// curl -X POST http://127.0.0.1:8080/delete/note --data `{"user_name": "some_name", "title": "some_title"}`
func (h *handler) DeleteUserNotes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body to get user's name
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var userNotesRequest internal.Note
	if err := json.Unmarshal(body, &userNotesRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// delete notes from goph-keeper storage
	if err = h.db.DeleteNotes(ctx, userNotesRequest); err != nil {
		message, status := parseUserError(userNotesRequest.UserName, err)
		http.Error(w, message, status)
		return
	}

	// response
	response := fmt.Sprintf("notes for user %q was successfully deleted", userNotesRequest.UserName)
	if userNotesRequest.Title != nil {
		response = fmt.Sprintf("notes for user %q with title %q was successfully deleted", userNotesRequest.UserName, *userNotesRequest.Title)
	}

	if _, err = io.WriteString(w, response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// UpdateUserNote is a method for updating note content and metadata
// for authorized user with provided note's title. Request body must contain user's name, note's title and new content. Metadata is optional.
// For example:
// curl -X POST http://127.0.0.1:8080/update/note --data `{"user_name": "some_name", "title": "some_title", "content": "new shopping list", "metadata": "some optional data"}`
func (h *handler) UpdateUserNote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var requestNote internal.Note
	if err := json.Unmarshal(buf.Bytes(), &requestNote); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if requestNote.Title == nil || requestNote.Content == nil {
		http.Error(w, "tile and content should not be empty", http.StatusBadRequest)
		return
	}

	// update note for user in goph-keeper storage
	if err := h.db.UpdateNote(ctx, requestNote); err != nil {
		message, status := parseUserError(requestNote.UserName, err)
		http.Error(w, message, status)
		return
	}

	// response
	if _, err := io.WriteString(w, fmt.Sprintf("updated note for user %q", requestNote.UserName)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// SaveCard is a method for saving provided bank card (bank name, number, cv, password and probably metadata)
// for authorized user. Request body must contain user's name, bank name, cv, password. Metadata is optional.
// For example:
// curl -X POST http://127.0.0.1:8080/save/card --data `{"user_name": "some_name", "bank_name": "alpha", "number":"1111222233334444", "cv": "123", "password": "3452", "metadata": "the card with a lot of money"}`
func (h *handler) SaveCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var requestCard internal.Card
	if err := json.Unmarshal(buf.Bytes(), &requestCard); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// save card in goph-keeper storage
	if err := h.db.SaveCard(ctx, requestCard); err != nil {
		message, status := parseUserError(requestCard.UserName, err)
		http.Error(w, message, status)
		return
	}

	// response
	if _, err := io.WriteString(w, fmt.Sprintf("saved card for user %q", requestCard.UserName)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetCard is a method for getting user's cards (bank names, numbers, cv, passwords and probably metadata)
// for provided authorized user. Request body must contain user's name. Bank name and number are optional parameters.
// For example: curl -X POST http://127.0.0.1:8080/get/card --data `{"user_name": "some_name", "bank_name":"tinkofff" ,"number": "1111222233334444"}`
func (h *handler) GetCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body to get user's name
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var cardRequest internal.Card
	if err = json.Unmarshal(body, &cardRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// get user cards from goph-keeper storage
	cards, err := h.db.GetCard(ctx, cardRequest)
	if err != nil {
		message, status := parseUserError(cardRequest.UserName, err)
		http.Error(w, message, status)
		return
	}

	// response
	cardsResponse, err := json.Marshal(cards)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err = io.WriteString(w, string(cardsResponse)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// DeleteCard is a method for deleting cards for provided user.
// Request body must contain user's name, bank name and card number are optional.
// For example:
// curl -X POST http://127.0.0.1:8080/delete/card --data `{"user_name": "some_name", "bank_name": "tinkoff", "number": "1111222233334444"}`
func (h *handler) DeleteCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	ctx := context.Background()
	// parse body to get user's name
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var cardRequest internal.Card
	if err = json.Unmarshal(body, &cardRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// delete card from goph-keeper storage
	if err = h.db.DeleteCards(ctx, cardRequest); err != nil {
		message, status := parseUserError(cardRequest.UserName, err)
		http.Error(w, message, status)
		return
	}

	// response
	response := fmt.Sprintf("cards for user %q was successfully deleted", cardRequest.UserName)
	if cardRequest.BankName != nil {
		response = fmt.Sprintf("cards of %q bank for user %q was successfully deleted", *cardRequest.BankName, cardRequest.UserName)
	} else if cardRequest.Number != nil {
		response = fmt.Sprintf("cards with number %q for user %q was successfully deleted", *cardRequest.Number, cardRequest.UserName)
	}

	if _, err = io.WriteString(w, response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// BasicAuth is a method for checking if current user is authorized.
func (h *handler) BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// parse body
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var user internal.Credentials
		if err := json.Unmarshal(buf.Bytes(), &user); err != nil {
			log.Println("BasicAuth")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// check cookies
		if h.cookies[user.UserName] == "" {
			http.Error(w, fmt.Sprintf("user %q is not authorized", user.UserName), http.StatusUnauthorized)
			return
		}

		// check token
		tkn, err := extractJwtToken(h.cookies[user.UserName])
		if err != nil {
			message, status := parseUserError(user.UserName, err)
			http.Error(w, message, status)
			return
		}
		if !tkn.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		w.Header().Add("Authorization", h.cookies[user.UserName])
		r.Body = io.NopCloser(bytes.NewBuffer(buf.Bytes()))
		next.ServeHTTP(w, r)
	})
}
