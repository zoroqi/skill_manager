package tui

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
	"os"
	"sort"
	"strings"
)

type dataLoadedMsg struct {
	projectSkills []ResolvedSkill
	globalSkills  []ResolvedSkill
	registerItems []RegisterItem
	groupItems    []GroupItem
	status        string
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func (m Model) Init() tea.Cmd {
	return m.loadData("")
}

func (m Model) loadData(status string) tea.Cmd {
	return func() tea.Msg {
		cwd, _ := os.Getwd()

		// Load Project
		pCtx, pSkills, pGroups, _ := m.app.GetStatus(cwd, false)
		pResolved := m.resolveStatus(cwd, pCtx, pSkills, pGroups)

		// Load Global
		gCtx, gSkills, gGroups, _ := m.app.GetStatus(cwd, true)
		gResolved := m.resolveStatus(cwd, gCtx, gSkills, gGroups)

		// Load Registries
		regs, _ := m.app.RegisterList()
		allSkills, _ := m.app.ListSkills()

		var items []RegisterItem
		for _, r := range regs {
			items = append(items, RegisterItem{
				IsRegistry: true,
				Name:       r.Name,
				URL:        r.URL,
			})
			for _, s := range allSkills {
				if s.RegistryName == r.Name {
					skillCopy := s
					items = append(items, RegisterItem{
						IsRegistry: false,
						Name:       s.Name,
						Skill:      &skillCopy,
					})
				}
			}
		}

		// Load Groups
		groupsCfg, _ := m.app.ListGroups()

		var grpItems []GroupItem
		for _, g := range groupsCfg.Groups {
			groupCopy := g
			grpItems = append(grpItems, GroupItem{
				IsGroup: true,
				Name:    g.Name,
				Summary: g.Description,
				Group:   &groupCopy,
			})
			for _, sid := range g.SkillIDs {
				for _, s := range allSkills {
					if s.ID == sid {
						skillCopy := s
						grpItems = append(grpItems, GroupItem{
							IsGroup:     false,
							Name:        s.Name,
							Skill:       &skillCopy,
							ParentGroup: g.Name,
						})
						break
					}
				}
			}
		}

		return dataLoadedMsg{
			projectSkills: pResolved,
			globalSkills:  gResolved,
			registerItems: items,
			groupItems:    grpItems,
			status:        status,
		}
	}
}

func (m Model) resolveStatus(dir string, ctxCfg internal.ContextConfig, skills []internal.Skill, groups []internal.Group) []ResolvedSkill {
	skillMap := make(map[string]internal.Skill)
	for _, s := range skills {
		skillMap[s.ID] = s
	}

	groupMap := make(map[string]internal.Group)
	for _, g := range groups {
		groupMap[g.Name] = g
	}

	existingAgents := store.FindExistingAgents(dir)

	inactive := make(map[string]map[string]bool)
	for _, usage := range ctxCfg.InactiveSkills {
		if inactive[usage.Name] == nil {
			inactive[usage.Name] = make(map[string]bool)
		}
		if len(usage.Agents) == 0 {
			inactive[usage.Name][""] = true
		} else {
			for _, a := range usage.Agents {
				inactive[usage.Name][a] = true
			}
		}
	}
	isInactive := func(name, agent string) bool {
		m := inactive[name]
		if m == nil {
			return false
		}
		return m[""] || m[agent]
	}

	effectiveID := make(map[string]map[string]string)
	idSources := make(map[string]map[string]map[string]bool)

	setEffective := func(agent, skillName, id, source string) {
		if effectiveID[agent] == nil {
			effectiveID[agent] = make(map[string]string)
		}
		if _, exists := effectiveID[agent][skillName]; !exists {
			effectiveID[agent][skillName] = id
		}
		if effectiveID[agent][skillName] == id {
			if idSources[agent] == nil {
				idSources[agent] = make(map[string]map[string]bool)
			}
			if idSources[agent][id] == nil {
				idSources[agent][id] = make(map[string]bool)
			}
			idSources[agent][id][source] = true
		}
	}

	for _, usage := range ctxCfg.ActiveSkills {
		skill, ok := skillMap[usage.ID]
		if !ok {
			continue
		}
		agents := usage.Agents
		if len(agents) == 0 {
			agents = existingAgents
		}
		for _, a := range agents {
			setEffective(a, skill.Name, skill.ID, "user")
		}
	}

	sortedGroups := make([]internal.GroupUsage, len(ctxCfg.ActiveGroups))
	copy(sortedGroups, ctxCfg.ActiveGroups)
	sort.Slice(sortedGroups, func(i, j int) bool {
		return sortedGroups[i].Name < sortedGroups[j].Name
	})

	for _, usage := range sortedGroups {
		group, ok := groupMap[usage.Name]
		if !ok {
			continue
		}
		agents := usage.Agents
		if len(agents) == 0 {
			agents = existingAgents
		}

		for _, skillID := range group.SkillIDs {
			skill, ok := skillMap[skillID]
			if !ok {
				continue
			}
			for _, a := range agents {
				if isInactive(skill.Name, a) {
					continue
				}
				setEffective(a, skill.Name, skillID, usage.Name)
			}
		}
	}

	type displaySkill struct {
		Skill   internal.Skill
		Agents  map[string]bool
		Sources map[string]bool
	}
	activeDisplay := make(map[string]*displaySkill)

	for agent, sMap := range effectiveID {
		for _, id := range sMap {
			if activeDisplay[id] == nil {
				activeDisplay[id] = &displaySkill{
					Skill:   skillMap[id],
					Agents:  make(map[string]bool),
					Sources: make(map[string]bool),
				}
			}
			activeDisplay[id].Agents[agent] = true
			for s := range idSources[agent][id] {
				activeDisplay[id].Sources[s] = true
			}
		}
	}

	var sortedIDs []string
	for id := range activeDisplay {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Slice(sortedIDs, func(i, j int) bool {
		return activeDisplay[sortedIDs[i]].Skill.Name < activeDisplay[sortedIDs[j]].Skill.Name
	})

	var result []ResolvedSkill
	for _, id := range sortedIDs {
		d := activeDisplay[id]
		var agents []string
		for a := range d.Agents {
			agents = append(agents, a)
		}
		sort.Strings(agents)

		var sources []string
		for s := range d.Sources {
			sources = append(sources, s)
		}
		sort.Strings(sources)

		reg := d.Skill.RegistryName
		if reg == "" {
			reg = "local"
		}

		result = append(result, ResolvedSkill{
			ID:       d.Skill.Name,
			Register: reg,
			Agents:   strings.Join(agents, "/"),
			Source:   strings.Join(sources, ", "),
		})
	}
	return result
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == FormView {
			switch msg.String() {
			case "esc":
				m.state = DashboardView
				return m, nil
			case "up", "shift+tab":
				m.formFocus--
				if m.formFocus < 0 {
					m.formFocus = len(m.formInputs) - 1
				}
				cmds := make([]tea.Cmd, len(m.formInputs))
				for i := range m.formInputs {
					if i == m.formFocus {
						cmds[i] = m.formInputs[i].Focus()
					} else {
						m.formInputs[i].Blur()
					}
				}
				return m, tea.Batch(cmds...)
			case "down", "tab":
				m.formFocus++
				if m.formFocus >= len(m.formInputs) {
					m.formFocus = 0
				}
				cmds := make([]tea.Cmd, len(m.formInputs))
				for i := range m.formInputs {
					if i == m.formFocus {
						cmds[i] = m.formInputs[i].Focus()
					} else {
						m.formInputs[i].Blur()
					}
				}
				return m, tea.Batch(cmds...)
			case "enter":
				if m.formFocus == len(m.formInputs)-1 {
					return m, m.submitForm()
				}
				m.formFocus++
				cmds := make([]tea.Cmd, len(m.formInputs))
				for i := range m.formInputs {
					if i == m.formFocus {
						cmds[i] = m.formInputs[i].Focus()
					} else {
						m.formInputs[i].Blur()
					}
				}
				return m, tea.Batch(cmds...)
			}

			var cmd tea.Cmd
			m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
			return m, cmd
		}

		if m.state == DetailView {
			switch msg.String() {
			case "esc":
				m.state = DashboardView
				m.selectedSkill = nil
				m.selectedGroup = nil
				m.err = nil
				return m, nil
			case "p", "P":
				if m.selectedSkill != nil {
					m.active = ProjectPanel
					m.setupForm()
					m.formInputs[0].SetValue(m.selectedSkill.Name)
					if len(m.formInputs) > 1 {
						m.formInputs[1].SetValue(m.selectedSkill.RegistryName)
					}
					return m, nil
				} else if m.selectedGroup != nil {
					m.active = ProjectPanel
					m.setupForm()
					m.formInputs[0].SetValue(m.selectedGroup.Name)
					return m, nil
				}
			case "g", "G":
				if m.selectedSkill != nil {
					m.active = GlobalPanel
					m.setupForm()
					m.formInputs[0].SetValue(m.selectedSkill.Name)
					if len(m.formInputs) > 1 {
						m.formInputs[1].SetValue(m.selectedSkill.RegistryName)
					}
					return m, nil
				} else if m.selectedGroup != nil {
					m.active = GlobalPanel
					m.setupForm()
					m.formInputs[0].SetValue(m.selectedGroup.Name)
					return m, nil
				}
			case "u", "U":
				if m.selectedSkill != nil {
					var groups []list.Item
					for _, item := range m.groupItems {
						if item.IsGroup {
							groups = append(groups, item)
						}
					}
					m.groupSelectionList.SetItems(groups)
					m.state = GroupSelectionView
					return m, nil
				}
			}
			m.detailViewport, cmd = m.detailViewport.Update(msg)
			return m, cmd
		}

		if m.state == GroupSelectionView {
			switch msg.String() {
			case "esc":
				m.state = DetailView
				return m, nil
			case "enter":
				if item, ok := m.groupSelectionList.SelectedItem().(GroupItem); ok {
					return m, m.addSkillToGroup(item.Name)
				}
			}
			m.groupSelectionList, cmd = m.groupSelectionList.Update(msg)
			return m, cmd
		}

		if m.state == ConfirmView {
			switch msg.String() {
			case "y", "Y", "enter":
				m.state = DashboardView
				return m, m.deleteSelected()
			case "n", "N", "esc":
				m.state = DashboardView
				return m, nil
			}
			return m, nil
		}

		if m.state == DashboardView {
			if !m.isFiltering() {
				switch msg.String() {
				case "ctrl+c", "q":
					return m, tea.Quit
				case "tab":
					m.active = (m.active + 1) % 4
					return m, nil
				case "shift+tab":
					m.active = (m.active + 3) % 4
					return m, nil
				case "a":
					m.setupForm()
					return m, nil
				case "d":
					m.confirmMsg = m.getDeleteConfirmMsg()
					if m.confirmMsg != "" {
						m.state = ConfirmView
						return m, nil
					}
					return m, nil
				case "enter":
					if m.active == RegisterPanel {
						if item, ok := m.registerList.SelectedItem().(RegisterItem); ok {
							if !item.IsRegistry && item.Skill != nil {
								m.showDetailView(item.Skill)
								return m, nil
							}
						}
					} else if m.active == GroupPanel {
						if item, ok := m.groupList.SelectedItem().(GroupItem); ok {
							if item.IsGroup && item.Group != nil {
								m.showGroupDetailView(item.Group)
								return m, nil
							}
						}
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		halfW := m.width / 2
		halfH := m.height / 2

		m.projectList.SetSize(halfW, halfH)
		m.registerList.SetSize(halfW, halfH)
		m.globalList.SetSize(halfW, halfH-10)
		m.groupList.SetSize(halfW, halfH-10)
		m.groupSelectionList.SetSize(m.width-4, m.height-6)

		m.detailViewport.Width = m.width - 4
		m.detailViewport.Height = m.height - 6

	case dataLoadedMsg:
		m.projectSkills = msg.projectSkills
		m.globalSkills = msg.globalSkills
		m.registerItems = msg.registerItems
		m.groupItems = msg.groupItems
		m.statusMsg = msg.status

		m.projectList.SetItems(toItems(m.projectSkills))
		m.globalList.SetItems(toItems(m.globalSkills))
		m.registerList.SetItems(toItems(m.registerItems))
		m.groupList.SetItems(toItems(m.groupItems))

		m.state = DashboardView
		m.selectedSkill = nil
		m.selectedGroup = nil
		m.err = nil

	case errMsg:
		m.err = msg.err
	}

	if m.state == DashboardView {
		var cmd tea.Cmd
		switch m.active {
		case ProjectPanel:
			m.projectList, cmd = m.projectList.Update(msg)
		case GlobalPanel:
			m.globalList, cmd = m.globalList.Update(msg)
		case RegisterPanel:
			m.registerList, cmd = m.registerList.Update(msg)
		case GroupPanel:
			m.groupList, cmd = m.groupList.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m Model) isFiltering() bool {
	switch m.active {
	case ProjectPanel:
		return m.projectList.FilterState() == list.Filtering
	case GlobalPanel:
		return m.globalList.FilterState() == list.Filtering
	case RegisterPanel:
		return m.registerList.FilterState() == list.Filtering
	case GroupPanel:
		return m.groupList.FilterState() == list.Filtering
	}
	return false
}

func (m *Model) showDetailView(skill *internal.Skill) {
	b, _ := json.MarshalIndent(skill, "", "  ")
	m.selectedSkill = skill
	m.selectedGroup = nil
	m.err = nil

	m.detailViewport = viewport.New(m.width-4, m.height-6)
	m.detailViewport.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)
	m.detailViewport.SetContent(string(b))

	m.state = DetailView
}

func (m *Model) showGroupDetailView(group *internal.Group) {
	b, _ := json.MarshalIndent(group, "", "  ")
	m.selectedGroup = group
	m.selectedSkill = nil
	m.err = nil

	m.detailViewport = viewport.New(m.width-4, m.height-6)
	m.detailViewport.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)
	m.detailViewport.SetContent(string(b))

	m.state = DetailView
}

func (m *Model) setupForm() {
	m.state = FormView
	m.formFocus = 0
	m.err = nil
	m.formInputs = nil

	var fields []string
	switch m.active {
	case RegisterPanel:
		m.formTitle = "Add Registry"
		fields = []string{"Name", "URL"}
	case GroupPanel:
		m.formTitle = "Create Group"
		fields = []string{"Name", "Description"}
	case ProjectPanel, GlobalPanel:
		m.formTitle = "Add Skill/Group"
		fields = []string{"Name", "Registry (optional)", "Agents (comma separated)"}
	}

	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f
		if i == 0 {
			ti.Focus()
		}
		m.formInputs = append(m.formInputs, ti)
	}
}

func (m Model) submitForm() tea.Cmd {
	return func() tea.Msg {
		val := func(idx int) string {
			if idx < len(m.formInputs) {
				return m.formInputs[idx].Value()
			}
			return ""
		}

		var err error
		var successMsg string
		switch m.active {
		case RegisterPanel:
			err = m.app.RegisterAdd(val(1), val(0), "")
			successMsg = "Registry added successfully"
		case GroupPanel:
			err = m.app.CreateGroup(val(0), val(1))
			successMsg = "Group created successfully"
		case ProjectPanel, GlobalPanel:
			global := m.active == GlobalPanel
			cwd, _ := os.Getwd()
			name := val(0)
			reg := val(1)
			agents := []string{}
			if val(2) != "" {
				// simple comma split
				for s := range strings.SplitSeq(val(2), ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						agents = append(agents, s)
					}
				}
			}

			// We need to decide if it's a skill or group.
			// Try AddSkill first, if not found or something, well, AddGroup.
			err = m.app.AddSkill(cwd, name, reg, agents, global, false)
			if err != nil {
				// try group
				err = m.app.AddGroup(cwd, name, agents, global, false)
				if err == nil {
					successMsg = fmt.Sprintf("Group '%s' added successfully", name)
				}
			} else {
				successMsg = fmt.Sprintf("Skill '%s' added successfully", name)
			}
		}

		if err != nil {
			return errMsg{err}
		}
		return m.loadData(successMsg)()
	}
}

func (m Model) getDeleteConfirmMsg() string {
	switch m.active {
	case RegisterPanel:
		if item, ok := m.registerList.SelectedItem().(RegisterItem); ok {
			if item.IsRegistry {
				return fmt.Sprintf("Are you sure you want to remove registry '%s'?", item.Name)
			}
		}
	case GroupPanel:
		if item, ok := m.groupList.SelectedItem().(GroupItem); ok {
			if item.IsGroup {
				return fmt.Sprintf("Are you sure you want to delete group '%s'?", item.Name)
			} else if item.ParentGroup != "" {
				return fmt.Sprintf("Are you sure you want to remove skill '%s' from group '%s'?", item.Name, item.ParentGroup)
			}
		}
	case ProjectPanel:
		if item, ok := m.projectList.SelectedItem().(ResolvedSkill); ok {
			return fmt.Sprintf("Are you sure you want to remove skill '%s' from project?", item.ID)
		}
	case GlobalPanel:
		if item, ok := m.globalList.SelectedItem().(ResolvedSkill); ok {
			return fmt.Sprintf("Are you sure you want to remove skill '%s' from global?", item.ID)
		}
	}
	return ""
}

func (m Model) deleteSelected() tea.Cmd {
	return func() tea.Msg {
		var err error
		var successMsg string
		cwd, _ := os.Getwd()

		switch m.active {
		case RegisterPanel:
			if item, ok := m.registerList.SelectedItem().(RegisterItem); ok {
				if item.IsRegistry {
					err = m.app.RegisterRemove(item.Name, false)
					successMsg = fmt.Sprintf("Registry '%s' removed", item.Name)
				}
			}
		case GroupPanel:
			if item, ok := m.groupList.SelectedItem().(GroupItem); ok {
				if item.IsGroup {
					err = m.app.DeleteGroup(item.Name)
					successMsg = fmt.Sprintf("Group '%s' deleted", item.Name)
				} else if item.ParentGroup != "" {
					err = m.app.RemoveSkillFromGroup(item.ParentGroup, []string{item.Name})
					successMsg = fmt.Sprintf("Skill '%s' removed from group '%s'", item.Name, item.ParentGroup)
				}
			}
		case ProjectPanel:
			if item, ok := m.projectList.SelectedItem().(ResolvedSkill); ok {
				err = m.app.RemoveSkill(cwd, item.ID, nil, false)
				successMsg = fmt.Sprintf("Skill '%s' removed from project", item.ID)
			}
		case GlobalPanel:
			if item, ok := m.globalList.SelectedItem().(ResolvedSkill); ok {
				err = m.app.RemoveSkill(cwd, item.ID, nil, true)
				successMsg = fmt.Sprintf("Skill '%s' removed from global", item.ID)
			}
		}

		if err != nil {
			return errMsg{err}
		}
		return m.loadData(successMsg)()
	}
}

func (m Model) addSkillToGroup(groupName string) tea.Cmd {
	return func() tea.Msg {
		if m.selectedSkill == nil {
			return nil
		}
		err := m.app.AddSkillToGroup(groupName, []string{m.selectedSkill.Name}, m.selectedSkill.RegistryName)
		if err != nil {
			return errMsg{err}
		}
		return m.loadData(fmt.Sprintf("Skill '%s' added to group '%s'", m.selectedSkill.Name, groupName))()
	}
}

// Helper to convert slices to []list.Item
func toItems[T any](items []T) []list.Item {
	var listItems []list.Item
	for _, i := range items {
		if item, ok := any(i).(list.Item); ok {
			listItems = append(listItems, item)
		}
	}
	return listItems
}
