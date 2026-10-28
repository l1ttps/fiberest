package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"fiberest/internal/configs"
	"fiberest/internal/models"
	"fiberest/internal/modules/auth"
	"fiberest/internal/modules/users"
	"fiberest/internal/server"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

type fakeAuth struct{}

func (fakeAuth) FindSessionBySessionId(context.Context, string) (*models.Session, error) {
	return nil, auth.ErrSessionNotFound
}
func (fakeAuth) UpdateExpiresSession(context.Context, *models.Session) error { return nil }
func (fakeAuth) DeleteSession(context.Context, string) error                 { return nil }

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	cfg := &configs.Config{}
	cfg.App.Name = "Fiberest"
	cfg.App.Env = "test"

	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ErrorHandler: http_error.NewHandler(nil),
	})
	// Registration only touches handlers/guards, never the database.
	auth.RegisterRoutes(app, auth.NewController(auth.NewService(nil, cfg), cfg))
	users.UserRoutes(app, users.NewController(users.NewService(nil, cfg)), fakeAuth{})
	return app
}

func TestRouteTable(t *testing.T) {
	app := newTestApp(t)
	for _, r := range app.GetRoutes(true) {
		t.Logf("%-7s %s", r.Method, r.Path)
	}
}

func TestUnauthenticatedProtectedRoutes(t *testing.T) {
	app := newTestApp(t)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/users"},
		{http.MethodGet, "/users/1b4e28ba-2fa1-11d2-883f-0016d3cca427"},
		{http.MethodPatch, "/users/me"},
		{http.MethodGet, "/auth/session"},
		{http.MethodGet, "/auth/sessions"},
		{http.MethodPost, "/auth/logout"},
	} {
		resp, err := app.Test(httptest.NewRequest(tc.method, tc.path, nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "%s %s", tc.method, tc.path)
	}
}

func TestPublicRoutesAreNotGuarded(t *testing.T) {
	app := newTestApp(t)

	// Login is public: it fails on validation (400), not on auth (401).
	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/auth/login", nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Logout is not: without a session cookie the guard answers first.
	resp, err = app.Test(httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUnmatchedPathIs404NotJSONError(t *testing.T) {
	app := newTestApp(t)
	server.Register404Handler(app)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
