package validator

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidEmail reports whether s parses as an email address.
func ValidEmail(s string) bool {
	_, err := mail.ParseAddress(s)
	return err == nil
}

// ValidSlug reports whether s is a lowercase, dash-separated slug.
func ValidSlug(s string) bool {
	return slugRe.MatchString(s)
}

// Slugify converts an arbitrary string into a valid slug.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// Errors aggregates field validation errors.
type Errors map[string]string

func (e Errors) Add(field, msg string) { e[field] = msg }
func (e Errors) Any() bool             { return len(e) > 0 }
func (e Errors) Error() string {
	parts := make([]string, 0, len(e))
	for f, m := range e {
		parts = append(parts, fmt.Sprintf("%s: %s", f, m))
	}
	return strings.Join(parts, "; ")
}
