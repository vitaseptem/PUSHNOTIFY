package handlers

import (
	"errors"
	"net/http"
	"strings"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/astrazstudio/pushnotify/server/internal/middleware"
	"github.com/astrazstudio/pushnotify/server/internal/store"
	"github.com/astrazstudio/pushnotify/server/internal/token"
	"github.com/astrazstudio/pushnotify/server/pkg/crypto"
	"github.com/astrazstudio/pushnotify/server/pkg/validator"
)

type registerRequest struct {
	Email         string `json:"email"`
	FullName      string `json:"full_name"`
	Password      string `json:"password"`
	WorkspaceName string `json:"workspace_name"`
}

type authResponse struct {
	Token     string      `json:"token"`
	User      interface{} `json:"user"`
	Workspace interface{} `json:"workspace"`
}

// Register creates a user, their first workspace (with a fresh VAPID keypair)
// and returns an auth token.
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !validator.ValidEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if req.FullName == "" {
		writeError(w, http.StatusBadRequest, "full_name is required")
		return
	}

	if _, err := h.Store.GetUserByEmail(r.Context(), req.Email); err == nil {
		writeError(w, http.StatusConflict, "email already registered")
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash failed")
		return
	}
	user, err := h.Store.CreateUser(r.Context(), req.Email, req.FullName, hash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create user failed")
		return
	}

	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vapid generation failed")
		return
	}

	wsName := req.WorkspaceName
	if wsName == "" {
		wsName = req.FullName + "'s Workspace"
	}
	slug := uniqueSlug(wsName, req.Email)
	ws, err := h.Store.CreateWorkspace(r.Context(), user.ID, wsName, slug, pub, priv)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create workspace failed")
		return
	}

	tok, err := token.Issue(h.Cfg.JWT.Secret, user.ID, ws.ID, h.Cfg.JWT.Expiry)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue failed")
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Token: tok, User: user, Workspace: ws})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates a user and returns a token plus their first workspace.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := h.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil || !crypto.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !user.IsActive {
		writeError(w, http.StatusForbidden, "account disabled")
		return
	}

	ws, err := h.Store.GetWorkspaceByOwner(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "workspace lookup failed")
		return
	}
	tok, err := token.Issue(h.Cfg.JWT.Secret, user.ID, ws.ID, h.Cfg.JWT.Expiry)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue failed")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: tok, User: user, Workspace: ws})
}

// Refresh issues a fresh token for the currently authenticated identity.
func (h *Handlers) Refresh(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	workspaceID := middleware.WorkspaceID(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	tok, err := token.Issue(h.Cfg.JWT.Secret, userID, workspaceID, h.Cfg.JWT.Expiry)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": tok})
}

// Me returns the authenticated user and their workspace.
func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	user, err := h.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, statusForStoreError(err), "user not found")
		return
	}
	ws, err := h.Store.GetWorkspaceByID(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		writeError(w, statusForStoreError(err), "workspace not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": user, "workspace": ws})
}

// uniqueSlug builds a slug from the workspace name with a short random suffix
// derived from the email to reduce collisions.
func uniqueSlug(name, email string) string {
	base := validator.Slugify(name)
	if base == "" {
		base = "workspace"
	}
	suffix, _ := crypto.RandomSecret(3)
	suffix = strings.ToLower(strings.TrimRight(suffix, "-_"))
	if suffix == "" {
		suffix = "ws"
	}
	return base + "-" + suffix
}
