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

// SanitizeArgs redacts all argument values for queries that touch
// sensitive columns (e.g. password hashes) so they never reach logs.
func SanitizeArgs(query string, args []any) []any {
	if !strings.Contains(strings.ToLower(query), "password") {
		return args
	}
	redacted := make([]any, len(args))
	for i := range redacted {
		redacted[i] = "[REDACTED]"
	}
	return redacted
}
