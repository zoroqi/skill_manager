package tui

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
)

func TestUpdate_DetailView_QuickAdd(t *testing.T) {
	m := NewModel(nil)
	m.state = DetailView
	m.selectedSkill = &internal.Skill{
		Name:         "test-skill",
		RegistryName: "test-reg",
	}

	// Press 'p'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m2 := newModel.(Model)

	assert.Equal(t, FormView, m2.state)
	assert.Equal(t, ProjectPanel, m2.active)
	assert.Equal(t, "test-skill", m2.formInputs[0].Value())
	assert.Equal(t, "test-reg", m2.formInputs[1].Value())

	// Press 'g'
	m.state = DetailView
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m3 := newModel.(Model)

	assert.Equal(t, FormView, m3.state)
	assert.Equal(t, GlobalPanel, m3.active)
	assert.Equal(t, "test-skill", m3.formInputs[0].Value())
	assert.Equal(t, "test-reg", m3.formInputs[1].Value())

	// Press 'esc'
	m.state = DetailView
	m.err = assert.AnError
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m4 := newModel.(Model)
	assert.Equal(t, DashboardView, m4.state)
	assert.Nil(t, m4.selectedSkill)
	assert.Nil(t, m4.err)
}
