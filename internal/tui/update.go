package tui

import (
	"encoding/json"
	"github.com/charmbracelet/bubbletea"
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
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func (m Model) Init() tea.Cmd {
	return m.loadData()
}

func (m Model) loadData() tea.Cmd {
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
			grpItems = append(grpItems, GroupItem{
				IsGroup:     true,
				Name:        g.Name,
				Description: g.Description,
			})
			for _, sid := range g.SkillIDs {
				for _, s := range allSkills {
					if s.ID == sid {
						skillCopy := s
						grpItems = append(grpItems, GroupItem{
							IsGroup: false,
							Name:    s.Name,
							Skill:   &skillCopy,
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

func (m *Model) filterRegisterItems() {
	m.filteredItems = nil
	m.selectedRow[RegisterPanel] = 0
	if m.searchQuery == "" {
		m.filteredItems = append(m.filteredItems, m.registerItems...)
		return
	}

	query := strings.ToLower(m.searchQuery)
	for _, item := range m.registerItems {
		if strings.Contains(strings.ToLower(item.Name), query) {
			m.filteredItems = append(m.filteredItems, item)
		}
	}
}

func (m *Model) filterGroupItems() {
	m.filteredGroupItems = nil
	m.selectedRow[GroupPanel] = 0
	if m.searchQuery == "" {
		m.filteredGroupItems = append(m.filteredGroupItems, m.groupItems...)
		return
	}

	query := strings.ToLower(m.searchQuery)
	for _, item := range m.groupItems {
		if strings.Contains(strings.ToLower(item.Name), query) {
			m.filteredGroupItems = append(m.filteredGroupItems, item)
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.searchQuery = ""
				if m.active == RegisterPanel {
					m.filterRegisterItems()
				} else if m.active == GroupPanel {
					m.filterGroupItems()
				}
			case "enter":
				m.searchMode = false
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					if m.active == RegisterPanel {
						m.filterRegisterItems()
					} else if m.active == GroupPanel {
						m.filterGroupItems()
					}
				}
			default:
				if len(msg.String()) == 1 {
					m.searchQuery += msg.String()
					if m.active == RegisterPanel {
						m.filterRegisterItems()
					} else if m.active == GroupPanel {
						m.filterGroupItems()
					}
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "/":
			if m.state == DashboardView && (m.active == RegisterPanel || m.active == GroupPanel) {
				m.searchMode = true
				m.searchQuery = ""
				return m, nil
			}
		case "ctrl+c", "q":
			if m.state == DashboardView {
				return m, tea.Quit
			}
		case "esc":
			if m.state == FormView {
				m.state = DashboardView
				return m, nil
			}
			if m.state == DetailView {
				m.state = DashboardView
				return m, nil
			}
		case "tab":
			if m.state == DashboardView {
				m.active = (m.active + 1) % 4
			}
		case "shift+tab":
			if m.state == DashboardView {
				m.active = (m.active + 3) % 4
			}
		case "up", "k":
			if m.state == DashboardView {
				if m.selectedRow[m.active] > 0 {
					m.selectedRow[m.active]--
				}
			} else if m.state == FormView {
				if m.formFocus > 0 {
					m.formFocus--
				}
			}
		case "down", "j":
			if m.state == DashboardView {
				max := m.getPanelCount(m.active)
				if m.selectedRow[m.active] < max-1 {
					m.selectedRow[m.active]++
				}
			} else if m.state == FormView {
				if m.formFocus < len(m.formFields)-1 {
					m.formFocus++
				}
			}
		case "a":
			if m.state == DashboardView {
				m.setupForm()
				return m, nil
			}
		case "d":
			if m.state == DashboardView {
				return m, m.deleteSelected()
			}
		case "enter":
			if m.state == DashboardView && m.active == RegisterPanel {
				if len(m.filteredItems) > 0 && m.selectedRow[RegisterPanel] < len(m.filteredItems) {
					item := m.filteredItems[m.selectedRow[RegisterPanel]]
					if !item.IsRegistry && item.Skill != nil {
						b, _ := json.MarshalIndent(item.Skill, "", "  ")
						m.detailContent = string(b)
						m.state = DetailView
						return m, nil
					}
				}
			}
			if m.state == DashboardView && m.active == GroupPanel {
				if len(m.filteredGroupItems) > 0 && m.selectedRow[GroupPanel] < len(m.filteredGroupItems) {
					item := m.filteredGroupItems[m.selectedRow[GroupPanel]]
					if !item.IsGroup && item.Skill != nil {
						b, _ := json.MarshalIndent(item.Skill, "", "  ")
						m.detailContent = string(b)
						m.state = DetailView
						return m, nil
					}
				}
			}
			if m.state == FormView {
				return m, m.submitForm()
			}
		case "backspace":
			if m.state == FormView {
				if len(m.formValues[m.formFocus]) > 0 {
					m.formValues[m.formFocus] = m.formValues[m.formFocus][:len(m.formValues[m.formFocus])-1]
				}
			}
		default:
			if m.state == FormView {
				if len(msg.String()) == 1 {
					m.formValues[m.formFocus] += msg.String()
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case dataLoadedMsg:
		m.projectSkills = msg.projectSkills
		m.globalSkills = msg.globalSkills
		m.registerItems = msg.registerItems
		m.groupItems = msg.groupItems
		m.filterRegisterItems()
		m.filterGroupItems()
		m.state = DashboardView

	case errMsg:
		m.err = msg.err
	}

	return m, nil
}

func (m Model) getPanelCount(p Panel) int {
	switch p {
	case ProjectPanel:
		return len(m.projectSkills)
	case RegisterPanel:
		return len(m.filteredItems)
	case GlobalPanel:
		return len(m.globalSkills)
	case GroupPanel:
		return len(m.filteredGroupItems)
	}
	return 0
}

func (m *Model) setupForm() {
	m.state = FormView
	m.formFocus = 0
	switch m.active {
	case RegisterPanel:
		m.formTitle = "Add Registry"
		m.formFields = []string{"Name", "URL"}
		m.formValues = []string{"", ""}
	case GroupPanel:
		m.formTitle = "Create Group"
		m.formFields = []string{"Name", "Description"}
		m.formValues = []string{"", ""}
	case ProjectPanel, GlobalPanel:
		m.formTitle = "Add Skill/Group"
		m.formFields = []string{"Name", "Registry (optional)", "Agents (comma separated)"}
		m.formValues = []string{"", "", ""}
	}
}

func (m Model) submitForm() tea.Cmd {
	return func() tea.Msg {
		var err error
		switch m.active {
		case RegisterPanel:
			err = m.app.RegisterAdd(m.formValues[1], m.formValues[0], "")
		case GroupPanel:
			err = m.app.CreateGroup(m.formValues[0], m.formValues[1])
		case ProjectPanel, GlobalPanel:
			global := m.active == GlobalPanel
			cwd, _ := os.Getwd()
			name := m.formValues[0]
			reg := m.formValues[1]
			agents := []string{}
			if m.formValues[2] != "" {
				// simple comma split
				for s := range strings.SplitSeq(m.formValues[2], ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						agents = append(agents, s)
					}
				}
			}

			// We need to decide if it's a skill or group.
			// Try AddSkill first, if not found or something, well, AddGroup.
			// Actually, the CLI usually has separate commands.
			// Let's assume user wants to add skill.
			err = m.app.AddSkill(cwd, name, reg, agents, global, false)
			if err != nil {
				// try group
				err = m.app.AddGroup(cwd, name, agents, global, false)
			}
		}

		if err != nil {
			return errMsg{err}
		}
		return m.loadData()
	}
}

func (m Model) deleteSelected() tea.Cmd {
	return func() tea.Msg {
		var err error
		cwd, _ := os.Getwd()
		idx := m.selectedRow[m.active]

		switch m.active {
		case RegisterPanel:
			if idx < len(m.filteredItems) {
				item := m.filteredItems[idx]
				if item.IsRegistry {
					err = m.app.RegisterRemove(item.Name, false)
				}
			}
		case GroupPanel:
			if idx < len(m.filteredGroupItems) {
				item := m.filteredGroupItems[idx]
				if item.IsGroup {
					err = m.app.DeleteGroup(item.Name)
				}
			}
		case ProjectPanel, GlobalPanel:
			global := m.active == GlobalPanel
			if global {
				if idx < len(m.globalSkills) {
					err = m.app.RemoveSkill(cwd, m.globalSkills[idx].ID, nil, global)
				}
			} else {
				if idx < len(m.projectSkills) {
					err = m.app.RemoveSkill(cwd, m.projectSkills[idx].ID, nil, global)
				}
			}
		}

		if err != nil {
			return errMsg{err}
		}
		return m.loadData()
	}
}
