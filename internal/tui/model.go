package tui

import (
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/app"
)

type ViewState int

const (
	DashboardView ViewState = iota
	FormView
	DetailView
)

type Panel int

const (
	ProjectPanel Panel = iota
	RegisterPanel
	GlobalPanel
	GroupPanel
)

type ResolvedSkill struct {
	ID       string
	Register string
	Agents   string
	Source   string
}

type RegisterItem struct {
	IsRegistry bool
	Name       string
	URL        string
	Skill      *internal.Skill
}

type GroupItem struct {
	IsGroup     bool
	Name        string
	Description string
	Skill       *internal.Skill
}

type Model struct {
	app    *app.App
	state  ViewState
	active Panel
	width  int
	height int

	// Data for Dashboard
	projectSkills []ResolvedSkill
	globalSkills  []ResolvedSkill

	registerItems      []RegisterItem
	filteredItems      []RegisterItem
	groupItems         []GroupItem
	filteredGroupItems []GroupItem

	selectedRow [4]int // index for each panel

	// Form related state
	formTitle  string
	formFields []string
	formValues []string
	formFocus  int
	err        error

	searchMode    bool
	searchQuery   string
	detailContent string
}

func NewModel(a *app.App) Model {
	return Model{
		app:    a,
		state:  DashboardView,
		active: ProjectPanel,
	}
}
