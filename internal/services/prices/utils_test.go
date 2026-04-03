package prices

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNullFloat64_Zero(t *testing.T) {
	result := nullFloat64(0)
	assert.False(t, result.Valid)
}

func TestNullFloat64_Positive(t *testing.T) {
	result := nullFloat64(9.99)
	assert.True(t, result.Valid)
	assert.Equal(t, 9.99, result.Float64)
}

func TestNullFloat64_Negative(t *testing.T) {
	result := nullFloat64(-1.5)
	assert.True(t, result.Valid)
	assert.Equal(t, -1.5, result.Float64)
}

func TestNullFloat64_SmallPositive(t *testing.T) {
	result := nullFloat64(0.01)
	assert.True(t, result.Valid)
	assert.Equal(t, 0.01, result.Float64)
}
