package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/app"
	"strings"
)

type ViewState int

const (
	DashboardView ViewState = iota
	FormView
	DetailView
	GroupSelectionView
	ConfirmView
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
	Summary     string
	Skill       *internal.Skill
	Group       *internal.Group
	ParentGroup string // Name of the group this skill belongs to
}

type Model struct {
	app    *app.App
	state  ViewState
	active Panel
	width  int
	height int

	// Bubbles List models for Dashboard
	projectList        list.Model
	globalList         list.Model
	registerList       list.Model
	groupList          list.Model
	groupSelectionList list.Model

	// Data backing the lists (keep for easier data management/filtering)
	projectSkills []ResolvedSkill
	globalSkills  []ResolvedSkill
	registerItems []RegisterItem
	groupItems    []GroupItem

	// Form related state
	formTitle  string
	formInputs []textinput.Model
	formFocus  int
	err        error
	statusMsg  string
	confirmMsg string

	// Detail View State
	detailViewport viewport.Model
	selectedSkill  *internal.Skill
	selectedGroup  *internal.Group
}

func NewModel(a *app.App) Model {
	m := Model{
		app:    a,
		state:  DashboardView,
		active: ProjectPanel,
	}
	// Pre-initialize lists with empty items to avoid panics on SetSize before data is loaded
	empty := []list.Item{}
	delegate := skillDelegate{}

	// Helper to create and configure a list
	makeList := func(title string) list.Model {
		l := list.New(empty, delegate, 0, 0)
		l.Title = title
		l.Filter = func(term string, targets []string) []list.Rank {
			var ranks []list.Rank
			if term == "" {
				return ranks
			}

			term = strings.ToLower(term)
			for i, target := range targets {
				if strings.Contains(strings.ToLower(target), term) {
					ranks = append(ranks, list.Rank{Index: i})
				}
			}
			return ranks
		}
		l.SetShowHelp(false)
		l.SetShowStatusBar(false)
		l.SetShowTitle(false)
		return l
	}

	m.projectList = makeList("Project Skills")
	m.globalList = makeList("Global Skills")
	m.registerList = makeList("Registries & Skills")
	m.groupList = makeList("Groups & Skills")
	m.groupSelectionList = makeList("Select Group")

	m.detailViewport = viewport.New(0, 0)

	return m
}
