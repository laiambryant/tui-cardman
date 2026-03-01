// Package logging provides structured logging utilities for the application.
package logging

import "strings"

// SanitizeQuery removes newlines and tabs and collapses whitespace
// so queries are safe to log in single-line form.
func SanitizeQuery(q string) string {
	if q == "" {
		return q
	}
	s := strings.ReplaceAll(q, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}
