package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeQuery_Empty(t *testing.T) {
	assert.Equal(t, "", SanitizeQuery(""))
}

func TestSanitizeQuery_NoChange(t *testing.T) {
	assert.Equal(t, "SELECT * FROM cards", SanitizeQuery("SELECT * FROM cards"))
}

func TestSanitizeQuery_RemovesTabs(t *testing.T) {
	assert.Equal(t, "SELECT * FROM cards WHERE id = 1", SanitizeQuery("SELECT *\tFROM cards\tWHERE id = 1"))
}

func TestSanitizeQuery_RemovesNewlines(t *testing.T) {
	assert.Equal(t, "SELECT * FROM cards WHERE id = 1", SanitizeQuery("SELECT *\nFROM cards\nWHERE id = 1"))
}

func TestSanitizeQuery_CollapsesMultipleSpaces(t *testing.T) {
	assert.Equal(t, "SELECT * FROM cards", SanitizeQuery("SELECT  *   FROM    cards"))
}

func TestSanitizeQuery_MixedWhitespace(t *testing.T) {
	input := "SELECT *\n\tFROM\n\t\tcards\n\tWHERE\n\t\tid = 1"
	assert.Equal(t, "SELECT * FROM cards WHERE id = 1", SanitizeQuery(input))
}

func TestSanitizeQuery_LeadingTrailingWhitespace(t *testing.T) {
	assert.Equal(t, "SELECT 1", SanitizeQuery("  \t\n SELECT 1 \n\t  "))
}
