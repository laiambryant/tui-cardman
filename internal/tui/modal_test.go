package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestModalModel_InitialState(t *testing.T) {
	m := NewModalModel("Title", "Message", nil, nil, defaultStyleManager)
	assert.True(t, m.IsVisible())
	assert.Equal(t, 0, m.selected)
}

func TestModalModel_HideShow(t *testing.T) {
	m := NewModalModel("Title", "Message", nil, nil, defaultStyleManager)
	m = m.Hide()
	assert.False(t, m.IsVisible())
	m = m.Show()
	assert.True(t, m.IsVisible())
	assert.Equal(t, 0, m.selected, "Show() should reset selection to 0")
}

func TestModalModel_SetMessage(t *testing.T) {
	m := NewModalModel("Title", "Message", nil, nil, defaultStyleManager)
	m = m.SetMessage("New Title", "New Message")
	assert.Equal(t, "New Title", m.title)
	assert.Equal(t, "New Message", m.message)
}

func TestModalModel_SetDimensions(t *testing.T) {
	m := NewModalModel("Title", "Message", nil, nil, defaultStyleManager)
	m = m.SetDimensions(80, 24)
	assert.Equal(t, 80, m.width)
	assert.Equal(t, 24, m.height)
}

func TestModalModel_KeyNavigation(t *testing.T) {
	m := NewModalModel("Title", "Message", nil, nil, defaultStyleManager)
	assert.Equal(t, 0, m.selected)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	assert.Equal(t, 1, m.selected)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	assert.Equal(t, 0, m.selected)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.selected)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 0, m.selected)
}

func TestModalModel_EnterOnYes(t *testing.T) {
	confirmCalled := false
	onConfirm := func() tea.Cmd {
		confirmCalled = true
		return nil
	}
	m := NewModalModel("Title", "Message", onConfirm, nil, defaultStyleManager)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.selected)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.IsVisible())
	assert.True(t, confirmCalled)
}

func TestModalModel_EnterOnNo(t *testing.T) {
	cancelCalled := false
	onCancel := func() tea.Cmd {
		cancelCalled = true
		return nil
	}
	m := NewModalModel("Title", "Message", nil, onCancel, defaultStyleManager)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.IsVisible())
	assert.True(t, cancelCalled)
}

func TestModalModel_EscCancels(t *testing.T) {
	cancelCalled := false
	onCancel := func() tea.Cmd {
		cancelCalled = true
		return nil
	}
	m := NewModalModel("Title", "Message", nil, onCancel, defaultStyleManager)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.IsVisible())
	assert.True(t, cancelCalled)
}

func TestModalModel_QKeyCancels(t *testing.T) {
	cancelCalled := false
	onCancel := func() tea.Cmd {
		cancelCalled = true
		return nil
	}
	m := NewModalModel("Title", "Message", nil, onCancel, defaultStyleManager)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	assert.False(t, m.IsVisible())
	assert.True(t, cancelCalled)
}

func TestModalModel_UpdateWhenHidden(t *testing.T) {
	m := NewModalModel("Title", "Message", nil, nil, defaultStyleManager)
	m = m.Hide()
	orig := m
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Equal(t, orig.selected, m.selected)
}

func TestModalModel_ViewWhenHidden(t *testing.T) {
	m := NewModalModel("Title", "Message", nil, nil, defaultStyleManager)
	m = m.Hide()
	assert.Equal(t, "", m.View())
}

func TestModalModel_NilCallbacks(t *testing.T) {
	m := NewModalModel("Title", "Message", nil, nil, defaultStyleManager)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.False(t, m.IsVisible())
}
