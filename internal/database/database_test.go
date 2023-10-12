package database

import (
	"context"
	"crypto/aes"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kontik-pk/goph-keeper/internal"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestDb_Register(t *testing.T) {
	userLogin := "jon"
	userPassword := "winterfell"
	ctx := context.Background()

	t.Run("positive: new user", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into registered_users values").
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn: mockDB,
		}
		err = pg.Register(ctx, userLogin, userPassword)
		assert.NoError(t, err)
	})
	t.Run("negative: user exists", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into registered_users values").
			WillReturnError(ErrDublicateKey{Key: "registered_users_pkey"})

		pg := db{
			conn: mockDB,
		}
		err = pg.Register(ctx, userLogin, userPassword)
		assert.EqualError(t, err, ErrUserAlreadyExists.Error())
	})
	t.Run("negative: error while inserting", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into registered_users values").
			WillReturnError(errors.New("some insert error"))

		pg := db{
			conn: mockDB,
		}
		err = pg.Register(ctx, userLogin, userPassword)
		assert.EqualError(t, err, "error while executing register user query: some insert error")
	})
}

func TestDb_Login(t *testing.T) {
	userLogin := "sansa"
	userPassword := "ihateramsey"
	ctx := context.Background()

	t.Run("positive: successful login", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		hash, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
		assert.NoError(t, err)

		mock.ExpectQuery("select login, password from registered_users where login").
			WithArgs(userLogin).
			WillReturnRows(sqlmock.NewRows([]string{"login", "password"}).AddRow(userLogin, hash))

		pg := db{
			conn: mockDB,
		}
		err = pg.Login(ctx, userLogin, userPassword)
		assert.NoError(t, err)
	})
	t.Run("positive: no such user", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select login, password from registered_users where login").
			WithArgs(userLogin).
			WillReturnRows(sqlmock.NewRows([]string{"login", "password"}))

		pg := db{
			conn: mockDB,
		}
		err = pg.Login(ctx, userLogin, userPassword)
		assert.EqualError(t, err, ErrNoSuchUser.Error())
	})
	t.Run("positive: invalid credentials", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select login, password from registered_users where login").
			WithArgs(userLogin).
			WillReturnRows(sqlmock.NewRows([]string{"login", "password"}).AddRow(userLogin, "some-hash"))

		pg := db{
			conn: mockDB,
		}
		err = pg.Login(ctx, userLogin, userPassword)
		assert.EqualError(t, err, ErrInvalidCredentials.Error())
	})
	t.Run("positive: search query", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select login, password from registered_users where login").
			WithArgs(userLogin).
			WillReturnError(errors.New("search error"))

		pg := db{
			conn: mockDB,
		}
		err = pg.Login(ctx, userLogin, userPassword)
		assert.EqualError(t, err, "error while executing search query: search error")
	})
}

func TestDb_GetCredentials(t *testing.T) {
	key := "thisis32bitlongpassphraseimusing"
	c, _ := aes.NewCipher([]byte(key))
	userLogin := "arya"
	ctx := context.Background()

	t.Run("positive: success", func(t *testing.T) {
		expected := []internal.Credentials{
			{
				UserName: userLogin,
				Login:    Ptr("killer"),
				Password: Ptr("sansaisfreak"),
				Metadata: Ptr("bla bla password"),
			},
			{
				UserName: userLogin,
				Login:    Ptr("warrior"),
				Password: Ptr("valarmorgulis"),
				Metadata: Ptr("valar dohaeris"),
			},
			{
				UserName: userLogin,
				Login:    Ptr("avenger"),
				Password: Ptr("qwerty12"),
			},
		}

		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, login, password, metadata from credentials where user_name").
			WithArgs(userLogin).
			WillReturnRows(sqlmock.NewRows([]string{"user_name", "login", "password", "metadata"}).
				AddRow(userLogin, "killer", "zwkcxfLKNXGHrfgP", "bla bla password").
				AddRow(userLogin, "warrior", "ygke1+HOKWWSvfUNiQ==", "valar dohaeris").
				AddRow(userLogin, "avenger", "zR8XxOfadyU=", nil))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		creds, err := pg.GetCredentials(
			ctx,
			internal.Credentials{UserName: userLogin},
		)
		assert.NoError(t, err)
		assert.Equal(t, expected, creds)
	})
	t.Run("negative: no data for user", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, login, password, metadata from credentials where user_name").
			WithArgs(userLogin).
			WillReturnRows(sqlmock.NewRows([]string{"user_name", "login", "password", "metadata"}))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		_, err = pg.GetCredentials(ctx, internal.Credentials{
			UserName: userLogin,
		})
		assert.EqualError(t, err, ErrNoData.Error())
	})
	t.Run("negative: query error", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, login, password, metadata from credentials where user_name").
			WithArgs(userLogin).
			WillReturnError(errors.New("query error"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		_, err = pg.GetCredentials(ctx, internal.Credentials{
			UserName: userLogin,
		})
		assert.EqualError(t, err, "error while getting credentials for user \"arya\": query error")
	})
}

func TestDb_SaveCredentials(t *testing.T) {
	key := "thisis32bitlongpassphraseimusing"
	c, _ := aes.NewCipher([]byte(key))
	credentials := internal.Credentials{
		UserName: "tirion",
		Login:    Ptr("imp"),
		Password: Ptr("ilovewine"),
		Metadata: Ptr("password for brothels"),
	}
	ctx := context.Background()

	t.Run("positive: with metadata", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into credentials").
			WithArgs(credentials.UserName, credentials.Login, "1QQdwPbUL3mQ", credentials.Metadata).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		err = pg.SaveCredentials(ctx, credentials)
		assert.NoError(t, err)
	})

	t.Run("positive: without metadata", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into credentials").
			WithArgs(credentials.UserName, credentials.Login, "1QQdwPbUL3mQ", nil).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		credentials.Metadata = nil
		err = pg.SaveCredentials(ctx, credentials)
		assert.NoError(t, err)
	})
	t.Run("negative: exec error", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into credentials").
			WithArgs(credentials.UserName, credentials.Login, "1QQdwPbUL3mQ", nil).
			WillReturnError(errors.New("exec error"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		credentials.Metadata = nil
		err = pg.SaveCredentials(ctx, credentials)
		assert.EqualError(t, err, "error while saving credentials for user \"tirion\": exec error")
	})
}

func TestDb_DeleteCredentials(t *testing.T) {
	user := "daenerys"
	login := "motherofdragons"
	ctx := context.Background()

	t.Run("positive: no login", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from credentials").
			WithArgs(user).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteCredentials(ctx, internal.Credentials{
			UserName: user,
		})
		assert.NoError(t, err)
	})
	t.Run("positive: with login", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from credentials").
			WithArgs(user, &login).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteCredentials(ctx, internal.Credentials{
			UserName: user,
			Login:    &login,
		})
		assert.NoError(t, err)
	})
	t.Run("negative: no login", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from credentials").
			WithArgs(user).
			WillReturnError(errors.New("some error"))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteCredentials(ctx, internal.Credentials{
			UserName: user,
		})
		assert.EqualError(t, err, "error while deleting credentials for user \"daenerys\": some error")
	})
	t.Run("negative: with login", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from credentials").
			WithArgs(user, &login).
			WillReturnError(errors.New("some error"))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteCredentials(ctx, internal.Credentials{
			UserName: user,
			Login:    &login,
		})
		assert.EqualError(t, err, "error while deleting credentials for user \"daenerys\": some error")
	})
}

func TestDb_UpdateCredentials(t *testing.T) {
	key := "thisis32bitlongpassphraseimusing"
	c, _ := aes.NewCipher([]byte(key))
	credentials := internal.Credentials{
		UserName: "tirion",
		Login:    Ptr("imp"),
		Password: Ptr("ilovewine"),
		Metadata: Ptr("password for brothels"),
	}
	ctx := context.Background()

	t.Run("positive: with metadata", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("update credentials set password").
			WithArgs("1QQdwPbUL3mQ", credentials.Metadata, credentials.UserName, credentials.Login).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		err = pg.UpdateCredentials(ctx, credentials)
		assert.NoError(t, err)
	})

	t.Run("positive: without metadata", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("update credentials set password").
			WithArgs("1QQdwPbUL3mQ", nil, credentials.UserName, credentials.Login).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		credentials.Metadata = nil
		err = pg.UpdateCredentials(ctx, credentials)
		assert.NoError(t, err)
	})
	t.Run("negative: exec error", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("update credentials set password").
			WithArgs("1QQdwPbUL3mQ", nil, credentials.UserName, credentials.Login).
			WillReturnError(errors.New("exec error"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		credentials.Metadata = nil
		err = pg.UpdateCredentials(ctx, credentials)
		assert.EqualError(t, err, "error while updating credentials for user \"tirion\": exec error")
	})
}

func TestDb_SaveNote(t *testing.T) {
	key := "thisis32bitlongpassphraseimusing"
	c, _ := aes.NewCipher([]byte(key))
	note := internal.Note{
		UserName: "podric",
		Title:    Ptr("how to became a knight"),
		Content:  Ptr("some note content"),
		Metadata: Ptr("podric's best note"),
	}
	ctx := context.Background()

	t.Run("positive: with metadata", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into notes").
			WithArgs(note.UserName, *note.Title, "zwcf07PNKWOQ6PoLlO+3uU4=", note.Metadata).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		err = pg.SaveNote(ctx, note)
		assert.NoError(t, err)
	})

	t.Run("positive: without metadata", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into notes").
			WithArgs(note.UserName, *note.Title, "zwcf07PNKWOQ6PoLlO+3uU4=", nil).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		note.Metadata = nil
		err = pg.SaveNote(ctx, note)
		assert.NoError(t, err)
	})
	t.Run("negative: exec error", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into notes").
			WithArgs(note.UserName, note.Title, "zwcf07PNKWOQ6PoLlO+3uU4=", nil).
			WillReturnError(errors.New("exec error"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		note.Metadata = nil
		err = pg.SaveNote(ctx, note)
		assert.EqualError(t, err, "error while saving note for user \"podric\": exec error")
	})
}

func TestDb_GetNote(t *testing.T) {
	key := "thisis32bitlongpassphraseimusing"
	c, _ := aes.NewCipher([]byte(key))
	userLogin := "myrcella"
	ctx := context.Background()

	t.Run("positive: without title", func(t *testing.T) {
		expected := []internal.Note{
			{
				UserName: userLogin,
				Title:    Ptr("notes from dorne"),
				Content:  Ptr("some lovely notes"),
				Metadata: Ptr("love"),
			},
			{
				UserName: userLogin,
				Title:    Ptr("notes from king's landing"),
				Content:  Ptr("some not lovely notes"),
				Metadata: Ptr("my worst days"),
			},
			{
				UserName: userLogin,
				Title:    Ptr("my dear diary"),
				Content:  Ptr("personal notes"),
			},
		}

		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, title, content, metadata from notes where user_name").
			WithArgs(userLogin).
			WillReturnRows(sqlmock.NewRows([]string{"user_name", "title", "content", "metadata"}).
				AddRow(userLogin, "notes from dorne", "zwcf07PPKWGQpOBElPSmsjQ=", "love").
				AddRow(userLogin, "notes from king's landing", "zwcf07PNKWPVpPYSn/er95LFBKDJ", "my worst days").
				AddRow(userLogin, "my dear diary", "zA0AxfzNJ3vVpvYQn+g=", nil))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		creds, err := pg.GetNotes(ctx, internal.Note{
			UserName: userLogin,
		})
		assert.NoError(t, err)
		assert.Equal(t, expected, creds)
	})
	t.Run("positive: with title", func(t *testing.T) {
		expected := []internal.Note{
			{
				UserName: userLogin,
				Title:    Ptr("notes from dorne"),
				Content:  Ptr("some lovely notes"),
				Metadata: Ptr("love"),
			},
		}

		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, title, content, metadata from notes where user_name").
			WithArgs(userLogin, Ptr("notes from dorne")).
			WillReturnRows(sqlmock.NewRows([]string{"user_name", "title", "content", "metadata"}).
				AddRow(userLogin, "notes from dorne", "zwcf07PPKWGQpOBElPSmsjQ=", "love"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		creds, err := pg.GetNotes(ctx, internal.Note{
			UserName: userLogin,
			Title:    Ptr("notes from dorne"),
		})
		assert.NoError(t, err)
		assert.Equal(t, expected, creds)
	})
	t.Run("negative: no data for user", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, title, content, metadata from notes where user_name").
			WithArgs(userLogin).
			WillReturnRows(sqlmock.NewRows([]string{"user_name", "title", "content", "metadata"}))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		_, err = pg.GetNotes(ctx, internal.Note{
			UserName: userLogin,
		})
		assert.EqualError(t, err, ErrNoData.Error())
	})
	t.Run("negative: query error", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, title, content, metadata from notes where user_name").
			WithArgs(userLogin).
			WillReturnError(errors.New("query error"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		_, err = pg.GetNotes(ctx, internal.Note{
			UserName: userLogin,
		})
		assert.EqualError(t, err, "error while getting notes for user \"myrcella\": query error")
	})
}

func TestDb_DeleteNotes(t *testing.T) {
	user := "jorah"
	title := "how to became a stone"
	ctx := context.Background()

	t.Run("positive: no title", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from notes").
			WithArgs(user).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteNotes(ctx, internal.Note{
			UserName: user,
		})
		assert.NoError(t, err)
	})
	t.Run("positive: with title", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from notes").
			WithArgs(user, title).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteNotes(ctx, internal.Note{
			UserName: user,
			Title:    &title,
		})
		assert.NoError(t, err)
	})
	t.Run("negative: exec error", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from notes").
			WithArgs(user).
			WillReturnError(errors.New("some error"))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteNotes(ctx, internal.Note{
			UserName: user,
		})
		assert.EqualError(t, err, "error while deleting note for user \"jorah\": some error")
	})
}

func TestDb_UpdateNote(t *testing.T) {
	key := "thisis32bitlongpassphraseimusing"
	c, _ := aes.NewCipher([]byte(key))
	note := internal.Note{
		UserName: "varys",
		Title:    Ptr("shopping list"),
		Content:  Ptr("some clever things"),
		Metadata: Ptr("bla bla metadata"),
	}
	ctx := context.Background()

	t.Run("positive: with metadata", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("update notes set content").
			WithArgs("zwcf07PAKnKDretEjvO7uXwo", note.Metadata, note.UserName, *note.Title).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		err = pg.UpdateNote(ctx, note)
		assert.NoError(t, err)
	})

	t.Run("positive: without metadata", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("update notes set content").
			WithArgs("zwcf07PAKnKDretEjvO7uXwo", nil, note.UserName, note.Title).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		note.Metadata = nil
		err = pg.UpdateNote(ctx, note)
		assert.NoError(t, err)
	})
	t.Run("negative: exec error", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("update notes set content").
			WithArgs("zwcf07PAKnKDretEjvO7uXwo", nil, note.UserName, note.Title).
			WillReturnError(errors.New("exec error"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		note.Metadata = nil
		err = pg.UpdateNote(ctx, note)
		assert.EqualError(t, err, "error while updating note \"shopping list\" for user \"varys\": exec error")
	})
}

func TestDb_SaveCard(t *testing.T) {
	key := "thisis32bitlongpassphraseimusing"
	c, _ := aes.NewCipher([]byte(key))
	card := internal.Card{
		UserName: "Tywin",
		BankName: Ptr("tinkoff"),
		Number:   Ptr("1111222233334444"),
		CV:       Ptr("123"),
		Password: Ptr("legacy"),
		Metadata: Ptr("podric's best note"),
	}
	ctx := context.Background()

	t.Run("positive: with metadata", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into cards").
			WithArgs(card.UserName, *card.BankName, *card.Number, "jVpB", "0A0V1/Da", *card.Metadata).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		err = pg.SaveCard(ctx, card)
		assert.NoError(t, err)
	})

	t.Run("positive: without metadata", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into cards").
			WithArgs(card.UserName, *card.BankName, *card.Number, "jVpB", "0A0V1/Da", nil).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		card.Metadata = nil
		err = pg.SaveCard(ctx, card)
		assert.NoError(t, err)
	})
	t.Run("negative: exec error", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("insert into cards").
			WithArgs(card.UserName, *card.BankName, *card.Number, "jVpB", "0A0V1/Da", nil).
			WillReturnError(errors.New("exec error"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		card.Metadata = nil
		err = pg.SaveCard(ctx, card)
		assert.EqualError(t, err, "error while saving card data for user \"Tywin\": exec error")
	})
}

func TestDb_GetCard(t *testing.T) {
	key := "thisis32bitlongpassphraseimusing"
	c, _ := aes.NewCipher([]byte(key))
	userLogin := "theon"
	ctx := context.Background()

	t.Run("positive: without bank name and title", func(t *testing.T) {
		expected := []internal.Card{
			{
				UserName: userLogin,
				BankName: Ptr("alpha"),
				Number:   Ptr("9999333344446666"),
				CV:       Ptr("321"),
				Password: Ptr("ironborne"),
				Metadata: Ptr("red bank"),
			},
			{
				UserName: userLogin,
				BankName: Ptr("tinkoff"),
				Number:   Ptr("5555444433337777"),
				CV:       Ptr("954"),
				Password: Ptr("ramsey"),
				Metadata: Ptr("black bank"),
			},
			{
				UserName: userLogin,
				BankName: Ptr("sber"),
				Number:   Ptr("6666555544440000"),
				CV:       Ptr("492"),
				Password: Ptr("qwerty"),
			},
		}

		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, bank_name, number, cv, password, metadata from cards where user_name").
			WithArgs(userLogin).
			WillReturnRows(sqlmock.NewRows([]string{"user_name", "bank_name", "number", "cv", "password", "metadata"}).
				AddRow(userLogin, "alpha", "9999333344446666", "j1pD", "1Rod2PHMNHmQ", "red bank").
				AddRow(userLogin, "tinkoff", "5555444433337777", "hV1G", "zgkfxfba", "black bank").
				AddRow(userLogin, "sber", "6666555544440000", "iFFA", "zR8XxOfa", nil))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		creds, err := pg.GetCard(ctx, internal.Card{
			UserName: userLogin,
		})
		assert.NoError(t, err)
		assert.Equal(t, expected, creds)
	})
	t.Run("positive: with bank name", func(t *testing.T) {
		expected := []internal.Card{
			{
				UserName: userLogin,
				BankName: Ptr("alpha"),
				Number:   Ptr("9999333344446666"),
				CV:       Ptr("321"),
				Password: Ptr("ironborne"),
				Metadata: Ptr("red bank"),
			},
		}

		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, bank_name, number, cv, password, metadata from cards where user_name").
			WithArgs(userLogin, "alpha").
			WillReturnRows(sqlmock.NewRows([]string{"user_name", "bank_name", "number", "cv", "password", "metadata"}).
				AddRow(userLogin, "alpha", "9999333344446666", "j1pD", "1Rod2PHMNHmQ", "red bank"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		creds, err := pg.GetCard(ctx, internal.Card{
			UserName: userLogin,
			BankName: Ptr("alpha"),
		})
		assert.NoError(t, err)
		assert.Equal(t, expected, creds)
	})
	t.Run("positive: with number", func(t *testing.T) {
		expected := []internal.Card{
			{
				UserName: userLogin,
				BankName: Ptr("alpha"),
				Number:   Ptr("9999333344446666"),
				CV:       Ptr("321"),
				Password: Ptr("ironborne"),
				Metadata: Ptr("red bank"),
			},
		}

		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, bank_name, number, cv, password, metadata from cards where user_name").
			WithArgs(userLogin, "9999333344446666").
			WillReturnRows(sqlmock.NewRows([]string{"user_name", "bank_name", "number", "cv", "password", "metadata"}).
				AddRow(userLogin, "alpha", "9999333344446666", "j1pD", "1Rod2PHMNHmQ", "red bank"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		creds, err := pg.GetCard(ctx, internal.Card{
			UserName: userLogin,
			Number:   Ptr("9999333344446666"),
		})
		assert.NoError(t, err)
		assert.Equal(t, expected, creds)
	})
	t.Run("positive: with bank name and number", func(t *testing.T) {
		expected := []internal.Card{
			{
				UserName: userLogin,
				BankName: Ptr("alpha"),
				Number:   Ptr("9999333344446666"),
				CV:       Ptr("321"),
				Password: Ptr("ironborne"),
				Metadata: Ptr("red bank"),
			},
		}

		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, bank_name, number, cv, password, metadata from cards where user_name").
			WithArgs(userLogin, "alpha", "9999333344446666").
			WillReturnRows(sqlmock.NewRows([]string{"user_name", "bank_name", "number", "cv", "password", "metadata"}).
				AddRow(userLogin, "alpha", "9999333344446666", "j1pD", "1Rod2PHMNHmQ", "red bank"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		creds, err := pg.GetCard(ctx, internal.Card{
			UserName: userLogin,
			Number:   Ptr("9999333344446666"),
			BankName: Ptr("alpha"),
		})
		assert.NoError(t, err)
		assert.Equal(t, expected, creds)
	})
	t.Run("negative: no data for user", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, bank_name, number, cv, password, metadata from cards where user_name").
			WithArgs(userLogin).
			WillReturnRows(sqlmock.NewRows([]string{"user_name", "bank_name", "number", "cv", "password", "metadata"}))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		_, err = pg.GetCard(ctx, internal.Card{
			UserName: userLogin,
		})
		assert.EqualError(t, err, ErrNoData.Error())
	})
	t.Run("negative: query error", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectQuery("select user_name, bank_name, number, cv, password, metadata from cards where user_name").
			WithArgs(userLogin).
			WillReturnError(errors.New("query error"))

		pg := db{
			conn:          mockDB,
			encriptionKey: key,
			dataCipher:    c,
		}
		_, err = pg.GetCard(ctx, internal.Card{
			UserName: userLogin,
		})
		assert.EqualError(t, err, "error while getting cards for user \"theon\": query error")
	})
}

func TestDb_DeleteCards(t *testing.T) {
	user := "tomen"
	bank := "bank of braavos"
	number := "7777333399991111"
	ctx := context.Background()

	t.Run("positive: delete all cards for user", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from cards").
			WithArgs(user).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteCards(ctx, internal.Card{
			UserName: user,
		})
		assert.NoError(t, err)
	})
	t.Run("positive: delete cards of provided bank for user", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from cards").
			WithArgs(user, &bank).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteCards(ctx, internal.Card{
			UserName: user,
			BankName: &bank,
		})
		assert.NoError(t, err)
	})
	t.Run("positive: delete cards with provided number for user", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from cards").
			WithArgs(user, &number).
			WillReturnResult(sqlmock.NewResult(0, 0))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteCards(ctx, internal.Card{
			UserName: user,
			Number:   &number,
		})
		assert.NoError(t, err)
	})
	t.Run("negative: exec error", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockDB.Close()

		mock.ExpectExec("delete from cards").
			WithArgs(user).
			WillReturnError(errors.New("some error"))

		pg := db{
			conn: mockDB,
		}
		err = pg.DeleteCards(ctx, internal.Card{
			UserName: user,
		})
		assert.EqualError(t, err, "error while deleting cards for user \"tomen\": some error")
	})
}

func Ptr(s string) *string {
	return &s
}
