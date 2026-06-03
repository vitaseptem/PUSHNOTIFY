package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

var variableRe = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// RenderedContent is a template rendered for a specific notification.
type RenderedContent struct {
	Subject string
	// Bodies maps channel name -> rendered body.
	Bodies map[string]string
}

// TemplateService renders templates with dynamic variables.
type TemplateService struct{}

// NewTemplateService constructs the service.
func NewTemplateService() *TemplateService { return &TemplateService{} }

// Render substitutes {{variable}} placeholders across every channel body and
// the subject. It fails if a template-declared required variable is missing.
func (s *TemplateService) Render(t *models.Template, vars map[string]string) (*RenderedContent, error) {
	if err := s.validateVars(t, vars); err != nil {
		return nil, err
	}
	rc := &RenderedContent{
		Subject: substitute(t.Subject, vars),
		Bodies:  make(map[string]string),
	}
	for _, ch := range t.Channels {
		rc.Bodies[ch] = substitute(t.BodyForChannel(ch), vars)
	}
	return rc, nil
}

func (s *TemplateService) validateVars(t *models.Template, vars map[string]string) error {
	var missing []string
	for _, required := range t.Variables {
		if _, ok := vars[required]; !ok {
			missing = append(missing, required)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required template variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

// substitute replaces all {{name}} occurrences with their values. Unknown
// placeholders are replaced with an empty string.
func substitute(s string, vars map[string]string) string {
	return variableRe.ReplaceAllStringFunc(s, func(match string) string {
		name := variableRe.FindStringSubmatch(match)[1]
		if v, ok := vars[name]; ok {
			return v
		}
		return ""
	})
}

// ExtractVariables returns the unique variable names referenced in a body.
func ExtractVariables(bodies ...string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, body := range bodies {
		for _, m := range variableRe.FindAllStringSubmatch(body, -1) {
			if _, ok := seen[m[1]]; !ok {
				seen[m[1]] = struct{}{}
				out = append(out, m[1])
			}
		}
	}
	return out
}
