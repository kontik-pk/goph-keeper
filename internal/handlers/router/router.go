package router

import (
	"github.com/go-chi/chi"
	"github.com/kontik-pk/goph-keeper/internal"
	"github.com/kontik-pk/goph-keeper/internal/handlers/handler"
	"go.uber.org/zap"
)

func New(db postgres, log *zap.SugaredLogger) *chi.Mux {
	httpHandler := handler.New(db, log)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Post("/auth/register", httpHandler.Register)
		r.Post("/auth/login", httpHandler.Login)
	})
	r.Group(func(r chi.Router) {
		r.Use(httpHandler.BasicAuth)
		r.Post("/save/credentials", httpHandler.SaveUserCredentials)
		r.Post("/delete/credentials", httpHandler.DeleteUserCredentials)
		r.Post("/get/credentials", httpHandler.GetUserCredentials)
		r.Post("/update/credentials", httpHandler.UpdateUserCredentials)

		r.Post("/save/note", httpHandler.SaveUserNote)
		r.Post("/delete/note", httpHandler.DeleteUserNotes)
		r.Post("/get/note", httpHandler.GetUserNote)
		r.Post("/update/note", httpHandler.UpdateUserNote)

		// no update option for bank cards managing
		r.Post("/save/card", httpHandler.SaveCard)
		r.Post("/delete/card", httpHandler.DeleteCard)
		r.Post("/get/card", httpHandler.GetCard)
	})

	return r
}

type postgres interface {
	SaveCredentials(credentialsRequest internal.Credentials) error
	GetCredentials(credentialsRequest internal.Credentials) ([]internal.Credentials, error)
	DeleteCredentials(credentialsRequest internal.Credentials) error
	UpdateCredentials(credentialsRequest internal.Credentials) error
	SaveNote(note internal.Note) error
	GetNotes(noteRequest internal.Note) ([]internal.Note, error)
	SaveCard(card internal.Card) error
	GetCard(cardRequest internal.Card) ([]internal.Card, error)
	DeleteNotes(noteRequest internal.Note) error
	UpdateNote(note internal.Note) error
	Register(login string, password string) error
	Login(login string, password string) error
	DeleteCards(cardRequest internal.Card) error
}
