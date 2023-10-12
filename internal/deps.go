package internal

import "context"

//go:generate mockery --disable-version-string --filename storage_mock.go --name Storage
type Storage interface {
	SaveCredentials(ctx context.Context, credentialsRequest Credentials) error
	GetCredentials(ctx context.Context, credentialsRequest Credentials) ([]Credentials, error)
	DeleteCredentials(ctx context.Context, credentialsRequest Credentials) error
	UpdateCredentials(ctx context.Context, credentials Credentials) error
	SaveNote(ctx context.Context, note Note) error
	GetNotes(ctx context.Context, noteRequest Note) ([]Note, error)
	DeleteNotes(ctx context.Context, noteRequest Note) error
	UpdateNote(ctx context.Context, note Note) error
	SaveCard(ctx context.Context, card Card) error
	GetCard(ctx context.Context, cardRequest Card) ([]Card, error)
	DeleteCards(ctx context.Context, cardRequest Card) error
	Register(ctx context.Context, login string, password string) error
	Login(ctx context.Context, login string, password string) error
	Close() error
}
