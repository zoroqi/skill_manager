package tui

import (
	"fmt"
	"strings"
)

const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorBold   = "\033[1m"
	ColorSelect = "\033[7m"
)

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	if m.state == FormView {
		return m.formView()
	}
	if m.state == DetailView {
		return m.detailView()
	}
	return m.dashboardView()
}

func (m Model) dashboardView() string {
	var b strings.Builder

	dashboardH := m.height
	if m.searchMode || m.searchQuery != "" {
		dashboardH--
	}

	halfW := m.width / 2
	halfH := dashboardH / 2

	// Render Top Row
	topL := m.renderPanel(ProjectPanel, halfW, halfH)
	topR := m.renderPanel(RegisterPanel, m.width-halfW, halfH)

	// Render Bottom Row
	botL := m.renderPanel(GlobalPanel, halfW, dashboardH-halfH)
	botR := m.renderPanel(GroupPanel, m.width-halfW, dashboardH-halfH)

	// Combine rows
	topLines := strings.Split(topL, "\n")
	rightLines := strings.Split(topR, "\n")
	for i := 0; i < halfH && i < len(topLines) && i < len(rightLines); i++ {
		b.WriteString(topLines[i])
		b.WriteString(rightLines[i])
		b.WriteString("\n")
	}

	botLines := strings.Split(botL, "\n")
	botRightLines := strings.Split(botR, "\n")
	for i := 0; i < (dashboardH-halfH) && i < len(botLines) && i < len(botRightLines); i++ {
		b.WriteString(botLines[i])
		b.WriteString(botRightLines[i])
		b.WriteString("\n")
	}

	if m.searchMode || m.searchQuery != "" {
		searchPrompt := "/" + m.searchQuery
		if m.searchMode {
			searchPrompt += "█"
		}
		b.WriteString(ColorGreen + searchPrompt + ColorReset)
	}

	return b.String()
}

type renderItem struct {
	text     string
	selected bool
}

func (m Model) renderPanel(p Panel, w, h int) string {
	var title string
	var header string
	var items []renderItem
	idx := m.selectedRow[p]

	switch p {
	case ProjectPanel:
		title = " Project Skills "
		header = fmt.Sprintf("%-30s %-20s %-20s %s", "Name", "Register", "Agent", "source")
		for i, s := range m.projectSkills {
			items = append(items, renderItem{
				text:     fmt.Sprintf("%-30s %-20s %-20s %s", truncate(s.ID, 30), truncate(s.Register, 20), truncate(s.Agents, 20), truncate(s.Source, 20)),
				selected: i == idx,
			})
		}
	case GlobalPanel:
		title = " Global Skills "
		header = fmt.Sprintf("%-30s %-20s %-20s %s", "Name", "Register", "Agent", "source")
		for i, s := range m.globalSkills {
			items = append(items, renderItem{
				text:     fmt.Sprintf("%-30s %-20s %-20s %s", truncate(s.ID, 30), truncate(s.Register, 20), truncate(s.Agents, 20), truncate(s.Source, 20)),
				selected: i == idx,
			})
		}
	case RegisterPanel:
		title = " Registries "
		// search query is now at the bottom
		header = fmt.Sprintf("%-20s %s", "Name", "skill_name")
		for i, item := range m.filteredItems {
			if item.IsRegistry {
				items = append(items, renderItem{
					text:     fmt.Sprintf("%-30s %s", truncate(item.Name, 30), truncate(item.URL, w)),
					selected: i == idx,
				})
			} else {
				items = append(items, renderItem{
					text:     fmt.Sprintf("%-30s %s", "", truncate(item.Name, w)),
					selected: i == idx,
				})
			}
		}
	case GroupPanel:
		title = " Groups "
		header = fmt.Sprintf("%-30s %s", "Name", "skill_name")
		for i, item := range m.filteredGroupItems {
			if item.IsGroup {
				items = append(items, renderItem{
					text:     fmt.Sprintf("%-30s %s", truncate(item.Name, 30), truncate(item.Description, w)),
					selected: i == idx,
				})
			} else {
				items = append(items, renderItem{
					text:     fmt.Sprintf("%-30s %s", "", truncate(item.Name, w)),
					selected: i == idx,
				})
			}
		}
	}

	borderColor := ColorReset
	if m.active == p {
		borderColor = ColorGreen
	}

	var res strings.Builder
	// Top border
	res.WriteString(borderColor + "┌" + title)
	res.WriteString(strings.Repeat("─", w-len(title)-2) + "┐" + ColorReset)
	res.WriteString("\n")

	// Header
	res.WriteString(borderColor + "│" + ColorReset)
	res.WriteString(ColorBold + truncate(header, w-2) + ColorReset)
	res.WriteString(strings.Repeat(" ", w-2-len(truncate(header, w-2))))
	res.WriteString(borderColor + "│" + ColorReset)
	res.WriteString("\n")

	// Separator
	res.WriteString(borderColor + "│" + strings.Repeat("┄", w-2) + "│" + ColorReset + "\n")

	// Content
	viewHeight := h - 4
	startIdx := 0

	// Better auto-scrolling logic: keep cursor around the middle
	if viewHeight > 0 && len(items) > viewHeight {
		halfView := viewHeight / 2
		if idx > halfView {
			startIdx = idx - halfView
		}
		// Prevent scrolling past the bottom
		if startIdx+viewHeight > len(items) {
			startIdx = len(items) - viewHeight
		}
	}

	for i := range viewHeight {
		res.WriteString(borderColor + "│" + ColorReset)
		content := ""
		itemIdx := startIdx + i
		if itemIdx < len(items) {
			item := items[itemIdx]
			displayLine := truncate(item.text, w-2)
			if item.selected && m.active == p {
				content = ColorSelect + displayLine + strings.Repeat(" ", w-2-len(displayLine)) + ColorReset
			} else {
				content = displayLine + strings.Repeat(" ", w-2-len(displayLine))
			}
		} else {
			content = strings.Repeat(" ", w-2)
		}
		res.WriteString(content)
		res.WriteString(borderColor + "│" + ColorReset)
		res.WriteString("\n")
	}

	// Bottom border
	res.WriteString(borderColor + "└" + strings.Repeat("─", w-2) + "┘" + ColorReset)

	return res.String()
}

func truncate(s string, w int) string {
	if len(s) > w {
		return s[:w-3] + "..."
	}
	return s
}

func (m Model) formView() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n  %s%s%s\n\n", ColorBold, m.formTitle, ColorReset))

	for i, field := range m.formFields {
		cursor := "  "
		if i == m.formFocus {
			cursor = ColorGreen + "> " + ColorReset
		}
		b.WriteString(fmt.Sprintf("%s%s: %s\n", cursor, field, m.formValues[i]))
	}

	b.WriteString("\n  (Enter to submit, Esc to cancel, Backspace to delete)\n")
	if m.err != nil {
		b.WriteString(fmt.Sprintf("\n  %sError: %v%s\n", ColorGreen, m.err, ColorReset))
	}

	return b.String()
}

func (m Model) detailView() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n  %sSkill Details%s\n\n", ColorBold, ColorReset))

	lines := strings.SplitSeq(m.detailContent, "\n")
	for line := range lines {
		b.WriteString(fmt.Sprintf("  %s\n", line))
	}

	b.WriteString("\n  (Esc to close)\n")
	return b.String()
}
