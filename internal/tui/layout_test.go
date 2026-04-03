package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitLines_Empty(t *testing.T) {
	assert.Equal(t, "", limitLines("", 5))
}

func TestLimitLines_ZeroMax(t *testing.T) {
	assert.Equal(t, "", limitLines("hello\nworld", 0))
}

func TestLimitLines_NegativeMax(t *testing.T) {
	assert.Equal(t, "", limitLines("hello\nworld", -1))
}

func TestLimitLines_WithinLimit(t *testing.T) {
	s := "line1\nline2\nline3"
	assert.Equal(t, s, limitLines(s, 5))
}

func TestLimitLines_ExactLimit(t *testing.T) {
	s := "line1\nline2\nline3"
	assert.Equal(t, s, limitLines(s, 3))
}

func TestLimitLines_ExceedsLimit(t *testing.T) {
	s := "line1\nline2\nline3\nline4\nline5"
	assert.Equal(t, "line1\nline2\nline3", limitLines(s, 3))
}

func TestLimitLines_SingleLine(t *testing.T) {
	assert.Equal(t, "hello", limitLines("hello", 1))
}

func TestTrimTrailingNewlines(t *testing.T) {
	assert.Equal(t, "hello", trimTrailingNewlines("hello\n\n\n"))
	assert.Equal(t, "hello", trimTrailingNewlines("hello"))
	assert.Equal(t, "", trimTrailingNewlines(""))
	assert.Equal(t, "", trimTrailingNewlines("\n\n"))
	assert.Equal(t, "a\nb", trimTrailingNewlines("a\nb\n"))
}

func TestJoinSections_AllPresent(t *testing.T) {
	assert.Equal(t, "header\nbody\nfooter", joinSections("header", "body", "footer"))
}

func TestJoinSections_EmptyParts(t *testing.T) {
	assert.Equal(t, "header\nfooter", joinSections("header", "", "footer"))
	assert.Equal(t, "body", joinSections("", "body", ""))
	assert.Equal(t, "", joinSections("", "", ""))
}

func TestResolveMinSectionHeight(t *testing.T) {
	assert.Equal(t, 1, resolveMinSectionHeight(1))
	assert.Equal(t, 1, resolveMinSectionHeight(5))
	assert.Equal(t, 2, resolveMinSectionHeight(6))
	assert.Equal(t, 2, resolveMinSectionHeight(8))
	assert.Equal(t, 3, resolveMinSectionHeight(9))
	assert.Equal(t, 3, resolveMinSectionHeight(100))
}

func TestReduceSectionHeights_NoDeficit(t *testing.T) {
	h, f := reduceSectionHeights(5, 5, 0, 3)
	assert.Equal(t, 5, h)
	assert.Equal(t, 5, f)
}

func TestReduceSectionHeights_SmallDeficit(t *testing.T) {
	h, f := reduceSectionHeights(6, 6, 2, 3)
	assert.Equal(t, 5, h)
	assert.Equal(t, 5, f)
}

func TestReduceSectionHeights_LargeDeficit(t *testing.T) {
	h, f := reduceSectionHeights(6, 6, 6, 3)
	assert.Equal(t, 3, h)
	assert.Equal(t, 3, f)
}

func TestReduceSectionHeights_MinAllowedClamp(t *testing.T) {
	h, f := reduceSectionHeights(4, 4, 10, 3)
	assert.GreaterOrEqual(t, h, 3)
	assert.GreaterOrEqual(t, f, 3)
}

func TestCalculateFrameLayout_ZeroDimensions(t *testing.T) {
	layout := calculateFrameLayout(1, 1, 0, 0)
	assert.Equal(t, 0, layout.Width)
	assert.Equal(t, 0, layout.Height)
	assert.Equal(t, 0, layout.BodyHeight)
}

func TestCalculateFrameLayout_NegativeWidth(t *testing.T) {
	layout := calculateFrameLayout(1, 1, -5, 20)
	assert.Equal(t, -5, layout.Width)
	assert.Equal(t, 20, layout.Height)
	assert.Equal(t, 0, layout.BodyHeight)
}

func TestCalculateFrameLayout_NormalDimensions(t *testing.T) {
	layout := calculateFrameLayout(2, 1, 80, 24)
	assert.Equal(t, 80, layout.Width)
	assert.Equal(t, 24, layout.Height)
	assert.Greater(t, layout.BodyHeight, 0)
	assert.Greater(t, layout.HeaderHeight, 0)
	assert.Greater(t, layout.FooterHeight, 0)
	assert.Equal(t, 24, layout.HeaderHeight+layout.BodyHeight+layout.FooterHeight)
}

func TestCalculateFrameLayout_ContentWidthPositive(t *testing.T) {
	layout := calculateFrameLayout(1, 1, 80, 24)
	assert.Greater(t, layout.ContentWidth, 0)
	assert.Less(t, layout.ContentWidth, 80)
}

func TestCalculateFrameLayout_VerySmallHeight(t *testing.T) {
	layout := calculateFrameLayout(1, 1, 80, 4)
	assert.GreaterOrEqual(t, layout.BodyHeight, 1)
}

func TestCalculateFrameLayout_ContentHeightsNonNegative(t *testing.T) {
	for height := 1; height <= 30; height++ {
		layout := calculateFrameLayout(1, 1, 80, height)
		assert.GreaterOrEqual(t, layout.HeaderContentHeight, 0, "height=%d", height)
		assert.GreaterOrEqual(t, layout.BodyContentHeight, 0, "height=%d", height)
		assert.GreaterOrEqual(t, layout.FooterContentHeight, 0, "height=%d", height)
	}
}
