package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/config"
	"github.com/astrazstudio/pushnotify/server/internal/models"
	"github.com/astrazstudio/pushnotify/server/pkg/crypto"
	"go.uber.org/zap"
)

func newTestHandlers(fs *fakeStore) *Handlers {
	return &Handlers{
		Cfg:   &config.Config{JWT: config.JWTConfig{Secret: "test-secret", Expiry: time.Hour}},
		Store: fs,
		Log:   zap.NewNop(),
	}
}

func doJSON(t *testing.T, h http.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestRegister_Success(t *testing.T) {
	fs := newFakeStore()
	h := newTestHandlers(fs)

	rec := doJSON(t, h.Register, http.MethodPost, "/api/v1/auth/register",
		`{"email":"a@b.com","full_name":"Ana","password":"supersecret"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Token     string           `json:"token"`
		User      models.User      `json:"user"`
		Workspace models.Workspace `json:"workspace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Token == "" || resp.User.Email != "a@b.com" || resp.Workspace.ID == "" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	fs := newFakeStore()
	fs.usersByEmail["a@b.com"] = &models.User{ID: "user_existing", Email: "a@b.com"}
	h := newTestHandlers(fs)

	rec := doJSON(t, h.Register, http.MethodPost, "/api/v1/auth/register",
		`{"email":"a@b.com","full_name":"Ana","password":"supersecret"}`)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "error") {
		t.Errorf("expected error field, got %s", rec.Body.String())
	}
}

func TestRegister_InvalidBody(t *testing.T) {
	h := newTestHandlers(newFakeStore())
	rec := doJSON(t, h.Register, http.MethodPost, "/api/v1/auth/register", `{not-json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	fs := newFakeStore()
	hash, _ := crypto.HashPassword("supersecret")
	fs.usersByEmail["a@b.com"] = &models.User{ID: "user_1", Email: "a@b.com", PasswordHash: hash, IsActive: true}
	fs.workspace = &models.Workspace{ID: "ws_1", OwnerID: "user_1"}
	h := newTestHandlers(fs)

	rec := doJSON(t, h.Login, http.MethodPost, "/api/v1/auth/login",
		`{"email":"a@b.com","password":"supersecret"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["token"] == "" || resp["token"] == nil {
		t.Errorf("expected token in response, got %v", resp)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	fs := newFakeStore()
	hash, _ := crypto.HashPassword("supersecret")
	fs.usersByEmail["a@b.com"] = &models.User{ID: "user_1", Email: "a@b.com", PasswordHash: hash, IsActive: true}
	h := newTestHandlers(fs)

	rec := doJSON(t, h.Login, http.MethodPost, "/api/v1/auth/login",
		`{"email":"a@b.com","password":"wrong"}`)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	h := newTestHandlers(newFakeStore())
	rec := doJSON(t, h.Login, http.MethodPost, "/api/v1/auth/login",
		`{"email":"ghost@b.com","password":"whatever"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (do not reveal user vs password)", rec.Code)
	}
}
