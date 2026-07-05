package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/laiambryant/tui-cardman/internal/model"
)

func newSizedTestModel() Model {
	return Model{styleManager: NewStyleManager(), width: 100, height: 30}
}

func TestMainView_RendersWithoutGames(t *testing.T) {
	m := newSizedTestModel()
	assert.NotEmpty(t, m.mainView())
}

func TestMainView_RendersWithGames(t *testing.T) {
	m := newSizedTestModel()
	m.cardGames = []model.CardGame{{ID: 1, Name: "Pokemon"}}
	assert.NotEmpty(t, m.mainView())
}

func TestSplashView_Renders(t *testing.T) {
	m := newSizedTestModel()
	assert.NotEmpty(t, m.splashView())
}

func TestLoginView_Renders(t *testing.T) {
	m := newSizedTestModel()
	m.initLoginInputs()
	assert.NotEmpty(t, m.loginView())
}

func TestRegisterView_Renders(t *testing.T) {
	m := newSizedTestModel()
	m.initRegisterInputs()
	assert.NotEmpty(t, m.registerView())
}

func TestLocalUserSetupView_Renders(t *testing.T) {
	m := newSizedTestModel()
	m.initLocalUserSetupInputs()
	assert.NotEmpty(t, m.localUserSetupView())
}

func TestCardGameMenuView_Renders(t *testing.T) {
	game := model.CardGame{ID: 1, Name: "Pokemon"}
	menu := NewCardGameMenuModel(&game, NewStyleManager(), nil)
	menu.width = 100
	menu.height = 30
	assert.NotEmpty(t, menu.View())
}

func TestModalView_Renders(t *testing.T) {
	modal := newModal("Title", "Message", nil, nil, NewStyleManager(), 80, 24)
	assert.NotEmpty(t, modal.View())
}

func TestRenderPanelHeading_FocusMarkers(t *testing.T) {
	sm := NewStyleManager()
	focused := renderPanelHeading(sm, "Panel", true)
	blurred := renderPanelHeading(sm, "Panel", false)
	assert.Contains(t, focused, "▍")
	assert.Contains(t, blurred, "▏")
}

func TestJoinHelp_UsesBulletSeparator(t *testing.T) {
	assert.Equal(t, "a • b", joinHelp("a", "b"))
}
