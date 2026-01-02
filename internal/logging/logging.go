package logging

import "strings"

// SanitizeQuery removes newlines and tabs and collapses whitespace
// so queries are safe to log in single-line form.
func SanitizeQuery(q string) string {
	if q == "" {
		return q
	}
	// replace tabs/newlines with spaces, then collapse consecutive spaces
	s := strings.ReplaceAll(q, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}
