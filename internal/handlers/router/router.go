package router

import (
	"github.com/go-chi/chi"
	"github.com/kontik-pk/goph-keeper/internal"
	"github.com/kontik-pk/goph-keeper/internal/handlers/handler"
	"go.uber.org/zap"
)

func New(db internal.Storage, log *zap.SugaredLogger) *chi.Mux {
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
