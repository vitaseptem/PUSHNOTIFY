package handlers

import (
	"net/http"

	"github.com/astrazstudio/pushnotify/server/internal/middleware"
	"github.com/astrazstudio/pushnotify/server/internal/models"
	"github.com/astrazstudio/pushnotify/server/internal/services"
	"github.com/astrazstudio/pushnotify/server/pkg/validator"
	"github.com/go-chi/chi/v5"
)

type templateRequest struct {
	Name          string   `json:"name"`
	Slug          string   `json:"slug"`
	Channels      []string `json:"channels"`
	Subject       string   `json:"subject"`
	BodyWebSocket string   `json:"body_websocket"`
	BodyWebPush   string   `json:"body_webpush"`
	BodyEmail     string   `json:"body_email"`
	BodyWhatsApp  string   `json:"body_whatsapp"`
	BodySMS       string   `json:"body_sms"`
	Variables     []string `json:"variables"`
}

func (req templateRequest) toModel(workspaceID string) *models.Template {
	slug := req.Slug
	if slug == "" {
		slug = validator.Slugify(req.Name)
	}
	vars := req.Variables
	if len(vars) == 0 {
		vars = services.ExtractVariables(req.Subject, req.BodyWebSocket, req.BodyWebPush,
			req.BodyEmail, req.BodyWhatsApp, req.BodySMS)
	}
	return &models.Template{
		WorkspaceID:   workspaceID,
		Name:          req.Name,
		Slug:          slug,
		Channels:      req.Channels,
		Subject:       req.Subject,
		BodyWebSocket: req.BodyWebSocket,
		BodyWebPush:   req.BodyWebPush,
		BodyEmail:     req.BodyEmail,
		BodyWhatsApp:  req.BodyWhatsApp,
		BodySMS:       req.BodySMS,
		Variables:     vars,
	}
}

// ListTemplates returns all templates for the workspace.
func (h *Handlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.ListTemplates(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// CreateTemplate creates a new template.
func (h *Handlers) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req templateRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || len(req.Channels) == 0 {
		writeError(w, http.StatusBadRequest, "name and channels are required")
		return
	}
	t := req.toModel(middleware.WorkspaceID(r.Context()))
	created, err := h.Store.CreateTemplate(r.Context(), t)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// GetTemplate returns a single template.
func (h *Handlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	t, err := h.Store.GetTemplateByID(r.Context(), middleware.WorkspaceID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, statusForStoreError(err), "template not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// UpdateTemplate updates an existing template.
func (h *Handlers) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	var req templateRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t := req.toModel(middleware.WorkspaceID(r.Context()))
	t.ID = chi.URLParam(r, "id")
	updated, err := h.Store.UpdateTemplate(r.Context(), t)
	if err != nil {
		writeError(w, statusForStoreError(err), "update failed")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// DeleteTemplate removes a template.
func (h *Handlers) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.DeleteTemplate(r.Context(), middleware.WorkspaceID(r.Context()), chi.URLParam(r, "id")); err != nil {
		writeError(w, statusForStoreError(err), "delete failed")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

type testTemplateRequest struct {
	Variables map[string]string `json:"variables"`
}

// TestTemplate renders a template with sample variables without sending.
func (h *Handlers) TestTemplate(w http.ResponseWriter, r *http.Request) {
	var req testTemplateRequest
	_ = decode(r, &req)
	t, err := h.Store.GetTemplateByID(r.Context(), middleware.WorkspaceID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, statusForStoreError(err), "template not found")
		return
	}
	rendered, err := h.Templates.Render(t, req.Variables)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"subject": rendered.Subject,
		"bodies":  rendered.Bodies,
	})
}
