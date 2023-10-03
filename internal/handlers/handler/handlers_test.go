package handler

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-resty/resty/v2"
	"github.com/kontik-pk/goph-keeper/internal"
	"github.com/kontik-pk/goph-keeper/internal/database"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_Login(t *testing.T) {
	userName := "jaime"
	password := "cersei"
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	testCases := []struct {
		name            string
		storageResponse error
		expectedCode    int
		cookies         bool
	}{
		{
			name:            "positive: success login",
			storageResponse: nil,
			expectedCode:    http.StatusOK,
			cookies:         true,
		},
		{
			name:            "negative: no such user",
			storageResponse: database.ErrNoSuchUser,
			expectedCode:    http.StatusUnauthorized,
			cookies:         false,
		},
		{
			name:            "negative: invalid credentials",
			storageResponse: database.ErrInvalidCredentials,
			expectedCode:    http.StatusUnauthorized,
			cookies:         false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedStorage := newMockStorage(t)
			mockedStorage.On("Login", userName, password).Return(tt.storageResponse)

			r := chi.NewRouter()
			h := New(mockedStorage, log)
			r.Post("/auth/login", h.Login)
			srv := httptest.NewServer(r)
			defer srv.Close()

			resp, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, userName, password)).
				Post(fmt.Sprintf("%s/auth/login", srv.URL))
			assert.NoError(t, err)
			assert.Equal(t, resp.StatusCode(), tt.expectedCode)
			if tt.cookies {
				assert.True(t, len(h.cookies) == 1)
				assert.True(t, h.cookies[userName] != "")
				assert.True(t, len(resp.Header().Get("Authorization")) > 1)
				assert.True(t, len(resp.Cookies()) == 1)
			}
		})
	}
	t.Run("negative: login is not provided", func(t *testing.T) {
		mockedStorage := newMockStorage(t)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/login", h.Login)
		srv := httptest.NewServer(r)
		defer srv.Close()

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"password": %q}`, password)).
			Post(fmt.Sprintf("%s/auth/login", srv.URL))
		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusBadRequest)
	})
	t.Run("negative: invalid body", func(t *testing.T) {
		mockedStorage := newMockStorage(t)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/login", h.Login)
		srv := httptest.NewServer(r)
		defer srv.Close()

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(`"password"`).
			Post(fmt.Sprintf("%s/auth/login", srv.URL))
		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusBadRequest)
	})
}

func TestHandler_Register(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	userName := "joffrey"
	password := "iamabadguy"

	testCases := []struct {
		name            string
		storageResponse error
		expectedCode    int
		cookies         bool
	}{
		{
			name:            "positive: success register",
			storageResponse: nil,
			expectedCode:    http.StatusOK,
			cookies:         true,
		},
		{
			name:            "negative: user exists",
			storageResponse: database.ErrUserAlreadyExists,
			expectedCode:    http.StatusConflict,
			cookies:         false,
		},
		{
			name:            "negative: db error",
			storageResponse: errors.New("some db error"),
			expectedCode:    http.StatusInternalServerError,
			cookies:         false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedStorage := newMockStorage(t)
			mockedStorage.On("Register", userName, password).Return(tt.storageResponse)

			r := chi.NewRouter()
			h := New(mockedStorage, log)
			r.Post("/auth/register", h.Register)
			srv := httptest.NewServer(r)
			defer srv.Close()

			resp, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, userName, password)).
				Post(fmt.Sprintf("%s/auth/register", srv.URL))
			assert.NoError(t, err)
			assert.Equal(t, resp.StatusCode(), tt.expectedCode)
			if tt.cookies {
				assert.True(t, len(h.cookies) == 1)
				assert.True(t, h.cookies[userName] != "")
				assert.True(t, len(resp.Header().Get("Authorization")) > 1)
				assert.True(t, len(resp.Cookies()) == 1)
			}
		})
	}
}

func TestHandler_GetUserCredentials(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	userName := "margaery"
	systemPassword := "rose"
	password := "ihatecersei"
	login := "queen"

	testCases := []struct {
		name                 string
		storageResponse      []internal.Credentials
		storageResponseError error
		expectedCode         int
		expectedBody         string
	}{
		{
			name: "positive: success getting credentials",
			storageResponse: []internal.Credentials{
				{
					UserName: userName,
					Login:    &login,
					Password: &password,
				},
			},
			expectedCode: http.StatusOK,
			expectedBody: fmt.Sprintf(`[{"user_name":%q,"login":%q,"password":%q}]`, userName, login, password),
		},
		{
			name:                 "negative: no data for user",
			storageResponse:      []internal.Credentials{},
			storageResponseError: database.ErrNoData,
			expectedCode:         http.StatusNoContent,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedStorage := newMockStorage(t)
			mockedStorage.On("Register", userName, systemPassword).Return(nil)
			mockedStorage.On("GetCredentials", internal.Credentials{UserName: userName}).Return(tt.storageResponse, tt.storageResponseError)

			r := chi.NewRouter()
			h := New(mockedStorage, log)
			r.Post("/auth/register", h.Register)
			r.Group(func(r chi.Router) {
				r.Post("/auth/register", h.Register)
			})
			r.Group(func(r chi.Router) {
				r.Use(h.BasicAuth)
				r.Post("/get/credentials", h.GetUserCredentials)
			})
			srv := httptest.NewServer(r)
			defer srv.Close()

			_, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, userName, systemPassword)).
				Post(fmt.Sprintf("%s/auth/register", srv.URL))
			assert.NoError(t, err)

			resp, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"user_name": %q}`, userName)).
				Post(fmt.Sprintf("%s/get/credentials", srv.URL))
			assert.NoError(t, err)
			assert.Equal(t, resp.StatusCode(), tt.expectedCode)
			assert.Equal(t, resp.String(), tt.expectedBody)
		})
	}

	t.Run("negative: invalid json", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", userName, password).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/get/credentials", h.GetUserCredentials)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, userName, password)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q`, userName)).
			Post(fmt.Sprintf("%s/get/credentials", srv.URL))
		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusBadRequest)
	})
	t.Run("negative: unauthorized user", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", userName, password).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/get/credentials", h.GetUserCredentials)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, userName, password)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(`{"user_name": "other"}`).
			Post(fmt.Sprintf("%s/get/credentials", srv.URL))
		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusUnauthorized)
	})
}

func TestHandler_SaveUserCredentials(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	systemName := "shae"
	systemPassword := "mylion"
	loginName := "badgirl"
	password := "money"
	metadata := "capital"

	testCases := []struct {
		name                 string
		storageResponseError error
		expectedCode         int
		expectedBody         string
	}{
		{
			name:         "positive: success saving credentials",
			expectedCode: http.StatusOK,
			expectedBody: `saved credentials for user "shae"`,
		},
		{
			name:                 "negative: saving error",
			expectedCode:         http.StatusInternalServerError,
			storageResponseError: errors.New("save error"),
			expectedBody:         `user "shae" request error : save error`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedStorage := newMockStorage(t)
			mockedStorage.On("Register", systemName, systemPassword).Return(nil)
			mockedStorage.On("SaveCredentials", internal.Credentials{UserName: systemName, Login: &loginName, Password: &password, Metadata: &metadata}).Return(tt.storageResponseError)

			r := chi.NewRouter()
			h := New(mockedStorage, log)
			r.Post("/auth/register", h.Register)
			r.Group(func(r chi.Router) {
				r.Post("/auth/register", h.Register)
			})
			r.Group(func(r chi.Router) {
				r.Use(h.BasicAuth)
				r.Post("/save/credentials", h.SaveUserCredentials)
			})
			srv := httptest.NewServer(r)
			defer srv.Close()

			_, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
				Post(fmt.Sprintf("%s/auth/register", srv.URL))
			assert.NoError(t, err)

			resp, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"user_name": %q, "login": %q, "password": %q, "metadata": %q}`, systemName, loginName, password, metadata)).
				Post(fmt.Sprintf("%s/save/credentials", srv.URL))

			assert.NoError(t, err)
			assert.Equal(t, resp.StatusCode(), tt.expectedCode)
			assert.Equal(t, resp.String(), tt.expectedBody)
		})
	}
	t.Run("negative: bad json", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/save/credentials", h.SaveUserCredentials)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q, "login1": %q}`, systemName, loginName)).
			Post(fmt.Sprintf("%s/save/credentials", srv.URL))

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusBadRequest)
	})
}

func TestHandler_DeleteUserCredentials(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	systemName := "missandei"
	systemPassword := "qwerty12"
	login := "blackgirl"

	t.Run("positive: with login", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)
		mockedStorage.On("DeleteCredentials", internal.Credentials{UserName: systemName, Login: &login}).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/delete/credentials", h.DeleteUserCredentials)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		request := fmt.Sprintf(`{"login": %q, "user_name": %q}`, login, systemName)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(request).
			Post(fmt.Sprintf("%s/delete/credentials", srv.URL))

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusOK)
		assert.Equal(t, resp.String(), "credentials for user \"missandei\" with login \"blackgirl\" was successfully deleted")
	})
	t.Run("positive: with no login", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)
		mockedStorage.On("DeleteCredentials", internal.Credentials{UserName: systemName}).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/delete/credentials", h.DeleteUserCredentials)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		request := fmt.Sprintf(`{"user_name": %q}`, systemName)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(request).
			Post(fmt.Sprintf("%s/delete/credentials", srv.URL))

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusOK)
		assert.Equal(t, resp.String(), "credentials for user \"missandei\" was successfully deleted")
	})
}

func TestHandler_UpdateUserCredentials(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	systemName := "shae"
	systemPassword := "mylion"
	loginName := "badgirl"
	password := "money"
	metadata := "capital"

	testCases := []struct {
		name                 string
		storageResponseError error
		expectedCode         int
		expectedBody         string
	}{
		{
			name:         "positive: success updating credentials",
			expectedCode: http.StatusOK,
			expectedBody: `updated credentials for user "shae"`,
		},
		{
			name:                 "negative: updating error",
			expectedCode:         http.StatusInternalServerError,
			storageResponseError: errors.New("update error"),
			expectedBody:         `user "shae" request error : update error`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedStorage := newMockStorage(t)
			mockedStorage.On("Register", systemName, systemPassword).Return(nil)
			mockedStorage.On("UpdateCredentials", internal.Credentials{UserName: systemName, Login: &loginName, Password: &password, Metadata: &metadata}).Return(tt.storageResponseError)

			r := chi.NewRouter()
			h := New(mockedStorage, log)
			r.Post("/auth/register", h.Register)
			r.Group(func(r chi.Router) {
				r.Post("/auth/register", h.Register)
			})
			r.Group(func(r chi.Router) {
				r.Use(h.BasicAuth)
				r.Post("/update/credentials", h.UpdateUserCredentials)
			})
			srv := httptest.NewServer(r)
			defer srv.Close()

			_, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
				Post(fmt.Sprintf("%s/auth/register", srv.URL))
			assert.NoError(t, err)

			resp, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"user_name": %q, "login": %q, "password": %q, "metadata": %q}`, systemName, loginName, password, metadata)).
				Post(fmt.Sprintf("%s/update/credentials", srv.URL))

			assert.NoError(t, err)
			assert.Equal(t, resp.StatusCode(), tt.expectedCode)
			assert.Equal(t, resp.String(), tt.expectedBody)
		})
	}
	t.Run("negative: bad json", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/update/credentials", h.UpdateUserCredentials)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q, "login1": %q}`, systemName, loginName)).
			Post(fmt.Sprintf("%s/update/credentials", srv.URL))

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusBadRequest)
	})
}

func TestHandler_SaveUserNote(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	systemName := "hound"
	systemPassword := "ihavebadbrother"
	loginName := "puppy"
	metadata := "no fire please"
	title := "some title"
	content := "some content"

	testCases := []struct {
		name                 string
		storageResponseError error
		expectedCode         int
		expectedBody         string
	}{
		{
			name:         "positive: success saving notest",
			expectedCode: http.StatusOK,
			expectedBody: `saved note for user "hound"`,
		},
		{
			name:                 "negative: saving error",
			expectedCode:         http.StatusInternalServerError,
			storageResponseError: errors.New("save error"),
			expectedBody:         `user "hound" request error : save error`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedStorage := newMockStorage(t)
			mockedStorage.On("Register", systemName, systemPassword).Return(nil)
			mockedStorage.On("SaveNote", internal.Note{UserName: systemName, Title: &title, Content: &content, Metadata: &metadata}).Return(tt.storageResponseError)

			r := chi.NewRouter()
			h := New(mockedStorage, log)
			r.Post("/auth/register", h.Register)
			r.Group(func(r chi.Router) {
				r.Post("/auth/register", h.Register)
			})
			r.Group(func(r chi.Router) {
				r.Use(h.BasicAuth)
				r.Post("/save/note", h.SaveUserNote)
			})
			srv := httptest.NewServer(r)
			defer srv.Close()

			_, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
				Post(fmt.Sprintf("%s/auth/register", srv.URL))
			assert.NoError(t, err)

			resp, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"user_name": %q, "title": %q, "content": %q, "metadata": %q}`, systemName, title, content, metadata)).
				Post(fmt.Sprintf("%s/save/note", srv.URL))

			assert.NoError(t, err)
			assert.Equal(t, resp.StatusCode(), tt.expectedCode)
			assert.Equal(t, resp.String(), tt.expectedBody)
		})
	}
	t.Run("negative: bad json", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/save/note", h.SaveUserNote)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q, "login1": %q}`, systemName, loginName)).
			Post(fmt.Sprintf("%s/save/note", srv.URL))

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusBadRequest)
	})
}

func TestHandler_GetUserNote(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	systemName := "hound"
	systemPassword := "ihavebadbrother"
	metadata := "no fire please"
	title := "some title"
	content := "some content"

	testCases := []struct {
		name                 string
		storageResponse      []internal.Note
		storageResponseError error
		expectedCode         int
		expectedBody         string
	}{
		{
			name: "positive: success getting credentials",
			storageResponse: []internal.Note{
				{
					UserName: systemName,
					Title:    &title,
					Content:  &content,
					Metadata: &metadata,
				},
			},
			expectedCode: http.StatusOK,
			expectedBody: fmt.Sprintf(`[{"user_name":%q,"title":%q,"content":%q,"metadata":%q}]`, systemName, title, content, metadata),
		},
		{
			name:                 "negative: no data for user",
			storageResponse:      []internal.Note{},
			storageResponseError: database.ErrNoData,
			expectedCode:         http.StatusNoContent,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedStorage := newMockStorage(t)
			mockedStorage.On("Register", systemName, systemPassword).Return(nil)
			mockedStorage.On("GetNotes", internal.Note{UserName: systemName}).Return(tt.storageResponse, tt.storageResponseError)

			r := chi.NewRouter()
			h := New(mockedStorage, log)
			r.Post("/auth/register", h.Register)
			r.Group(func(r chi.Router) {
				r.Post("/auth/register", h.Register)
			})
			r.Group(func(r chi.Router) {
				r.Use(h.BasicAuth)
				r.Post("/get/note", h.GetUserNote)
			})
			srv := httptest.NewServer(r)
			defer srv.Close()

			_, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
				Post(fmt.Sprintf("%s/auth/register", srv.URL))
			assert.NoError(t, err)

			resp, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"user_name": %q}`, systemName)).
				Post(fmt.Sprintf("%s/get/note", srv.URL))
			assert.NoError(t, err)
			assert.Equal(t, resp.StatusCode(), tt.expectedCode)
			assert.Equal(t, resp.String(), tt.expectedBody)
		})
	}

	t.Run("negative: invalid json", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/get/note", h.GetUserNote)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q`, systemName)).
			Post(fmt.Sprintf("%s/get/note", srv.URL))
		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusBadRequest)
	})
	t.Run("negative: unauthorized user", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/get/note", h.GetUserNote)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(`{"user_name": "other"}`).
			Post(fmt.Sprintf("%s/get/note", srv.URL))
		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusUnauthorized)
	})
}

func TestHandler_DeleteUserNotes(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	systemName := "missandei"
	systemPassword := "qwerty12"
	title := "some title"

	t.Run("positive: with title", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)
		mockedStorage.On("DeleteNotes", internal.Note{UserName: systemName, Title: &title}).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/delete/note", h.DeleteUserNotes)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q, "title":%q}`, systemName, title)).
			Post(fmt.Sprintf("%s/delete/note", srv.URL))

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusOK)
		assert.Equal(t, resp.String(), `notes for user "missandei" with title "some title" was successfully deleted`)
	})
	t.Run("positive: with no title", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)
		mockedStorage.On("DeleteNotes", internal.Note{UserName: systemName}).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/delete/note", h.DeleteUserNotes)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q}`, systemName)).
			Post(fmt.Sprintf("%s/delete/note", srv.URL))

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusOK)
		assert.Equal(t, resp.String(), "notes for user \"missandei\" was successfully deleted")
	})
}

func TestHandler_UpdateUserNote(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	systemName := "shae"
	systemPassword := "mylion"
	title := "some title"
	content := "some content"
	metadata := "capital"

	testCases := []struct {
		name                 string
		storageResponseError error
		expectedCode         int
		expectedBody         string
	}{
		{
			name:         "positive: success updating notes",
			expectedCode: http.StatusOK,
			expectedBody: `updated note for user "shae"`,
		},
		{
			name:                 "negative: updating error",
			expectedCode:         http.StatusInternalServerError,
			storageResponseError: errors.New("update error"),
			expectedBody:         `user "shae" request error : update error`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedStorage := newMockStorage(t)
			mockedStorage.On("Register", systemName, systemPassword).Return(nil)
			mockedStorage.On("UpdateNote", internal.Note{UserName: systemName, Title: &title, Content: &content, Metadata: &metadata}).Return(tt.storageResponseError)

			r := chi.NewRouter()
			h := New(mockedStorage, log)
			r.Post("/auth/register", h.Register)
			r.Group(func(r chi.Router) {
				r.Post("/auth/register", h.Register)
			})
			r.Group(func(r chi.Router) {
				r.Use(h.BasicAuth)
				r.Post("/update/note", h.UpdateUserNote)
			})
			srv := httptest.NewServer(r)
			defer srv.Close()

			_, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
				Post(fmt.Sprintf("%s/auth/register", srv.URL))
			assert.NoError(t, err)

			resp, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"user_name": %q, "title": %q, "content": %q, "metadata": %q}`, systemName, title, content, metadata)).
				Post(fmt.Sprintf("%s/update/note", srv.URL))

			assert.NoError(t, err)
			assert.Equal(t, resp.StatusCode(), tt.expectedCode)
			assert.Equal(t, resp.String(), tt.expectedBody)
		})
	}
	t.Run("negative: bad json", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/update/note", h.UpdateUserNote)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q, "login1": %q}`, systemName, title)).
			Post(fmt.Sprintf("%s/update/note", srv.URL))

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusBadRequest)
	})
}

func TestHandler_SaveCard(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	systemName := "hound"
	systemPassword := "ihavebadbrother"
	cv := "724"
	metadata := "green bank"
	bankName := "sber"
	password := "strong"
	number := "1111000033338888"

	testCases := []struct {
		name                 string
		storageResponseError error
		expectedCode         int
		expectedBody         string
	}{
		{
			name:         "positive: success saving сфкв",
			expectedCode: http.StatusOK,
			expectedBody: `saved card for user "hound"`,
		},
		{
			name:                 "negative: saving error",
			expectedCode:         http.StatusInternalServerError,
			storageResponseError: errors.New("save error"),
			expectedBody:         `user "hound" request error : save error`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedStorage := newMockStorage(t)
			mockedStorage.On("Register", systemName, systemPassword).Return(nil)
			mockedStorage.On("SaveCard", internal.Card{UserName: systemName, BankName: &bankName, Number: &number, CV: &cv, Password: &password, Metadata: &metadata}).Return(tt.storageResponseError)

			r := chi.NewRouter()
			h := New(mockedStorage, log)
			r.Post("/auth/register", h.Register)
			r.Group(func(r chi.Router) {
				r.Post("/auth/register", h.Register)
			})
			r.Group(func(r chi.Router) {
				r.Use(h.BasicAuth)
				r.Post("/save/card", h.SaveCard)
			})
			srv := httptest.NewServer(r)
			defer srv.Close()

			_, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
				Post(fmt.Sprintf("%s/auth/register", srv.URL))
			assert.NoError(t, err)

			resp, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"user_name": %q, "bank_name": %q, "number": %q,"cv":%q,"password":%q,"metadata": %q}`, systemName, bankName, number, cv, password, metadata)).
				Post(fmt.Sprintf("%s/save/card", srv.URL))

			assert.NoError(t, err)
			assert.Equal(t, resp.StatusCode(), tt.expectedCode)
			assert.Equal(t, resp.String(), tt.expectedBody)
		})
	}
}

func TestHandler_GetCard(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	systemName := "hound"
	systemPassword := "ihavebadbrother"
	metadata := "no fire please"
	bankName := "alpha"
	number := "0000888822227777"
	cv := "728"
	password := "strong"

	testCases := []struct {
		name                 string
		storageResponse      []internal.Card
		storageResponseError error
		expectedCode         int
		expectedBody         string
	}{
		{
			name: "positive: success getting credentials",
			storageResponse: []internal.Card{
				{
					UserName: systemName,
					BankName: &bankName,
					Number:   &number,
					CV:       &cv,
					Password: &password,
					Metadata: &metadata,
				},
			},
			expectedCode: http.StatusOK,
			expectedBody: fmt.Sprintf(`[{"user_name":%q,"bank_name":%q,"number":%q,"cv":%q,"password":%q,"metadata":%q}]`, systemName, bankName, number, cv, password, metadata),
		},
		{
			name:                 "negative: no data for user",
			storageResponse:      []internal.Card{},
			storageResponseError: database.ErrNoData,
			expectedCode:         http.StatusNoContent,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedStorage := newMockStorage(t)
			mockedStorage.On("Register", systemName, systemPassword).Return(nil)
			mockedStorage.On("GetCard", internal.Card{UserName: systemName}).Return(tt.storageResponse, tt.storageResponseError)

			r := chi.NewRouter()
			h := New(mockedStorage, log)
			r.Post("/auth/register", h.Register)
			r.Group(func(r chi.Router) {
				r.Post("/auth/register", h.Register)
			})
			r.Group(func(r chi.Router) {
				r.Use(h.BasicAuth)
				r.Post("/get/card", h.GetCard)
			})
			srv := httptest.NewServer(r)
			defer srv.Close()

			_, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
				Post(fmt.Sprintf("%s/auth/register", srv.URL))
			assert.NoError(t, err)

			resp, err := resty.New().R().
				SetHeader("content-type", "application/json").
				SetBody(fmt.Sprintf(`{"user_name": %q}`, systemName)).
				Post(fmt.Sprintf("%s/get/card", srv.URL))
			assert.NoError(t, err)
			assert.Equal(t, resp.StatusCode(), tt.expectedCode)
			assert.Equal(t, resp.String(), tt.expectedBody)
		})
	}

	t.Run("negative: invalid json", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/get/card", h.GetCard)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q`, systemName)).
			Post(fmt.Sprintf("%s/get/card", srv.URL))
		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusBadRequest)
	})
	t.Run("negative: unauthorized user", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/get/card", h.GetCard)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(`{"user_name": "other"}`).
			Post(fmt.Sprintf("%s/get/card", srv.URL))
		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusUnauthorized)
	})
}

func TestHandler_DeleteCard(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log := logger.Sugar()

	systemName := "hound"
	systemPassword := "ihavebadbrother"
	bankName := "alpha"
	number := "0000888822227777"

	t.Run("positive: with bank name", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)
		mockedStorage.On("DeleteCards", internal.Card{UserName: systemName, BankName: &bankName}).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/delete/card", h.DeleteCard)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q, "bank_name":%q}`, systemName, bankName)).
			Post(fmt.Sprintf("%s/delete/card", srv.URL))

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusOK)
		assert.Equal(t, resp.String(), `cards of "alpha" bank for user "hound" was successfully deleted`)
	})
	t.Run("positive: with no number", func(t *testing.T) {
		mockedStorage := newMockStorage(t)
		mockedStorage.On("Register", systemName, systemPassword).Return(nil)
		mockedStorage.On("DeleteCards", internal.Card{UserName: systemName, Number: &number}).Return(nil)

		r := chi.NewRouter()
		h := New(mockedStorage, log)
		r.Post("/auth/register", h.Register)
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", h.Register)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.BasicAuth)
			r.Post("/delete/card", h.DeleteCard)
		})
		srv := httptest.NewServer(r)
		defer srv.Close()

		_, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"login": %q, "password": %q}`, systemName, systemPassword)).
			Post(fmt.Sprintf("%s/auth/register", srv.URL))
		assert.NoError(t, err)

		resp, err := resty.New().R().
			SetHeader("content-type", "application/json").
			SetBody(fmt.Sprintf(`{"user_name": %q, "number":%q}`, systemName, number)).
			Post(fmt.Sprintf("%s/delete/card", srv.URL))

		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode(), http.StatusOK)
		assert.Equal(t, resp.String(), "cards with number \"0000888822227777\" for user \"hound\" was successfully deleted")
	})
}
