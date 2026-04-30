package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var (
	boldStyle   = lipgloss.NewStyle().Bold(true)
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	activeStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("2"))
	inactiveStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")). // White for better visibility
			Bold(true).
			PaddingLeft(4)
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
	if m.state == GroupSelectionView {
		return m.groupSelectionView()
	}
	if m.state == ConfirmView {
		return m.confirmView()
	}
	return m.dashboardView()
}

func (m Model) dashboardView() string {
	// Account for borders and headers
	halfW := m.width / 2
	halfH := m.height / 2
	m.projectList.SetSize(halfW, halfH)
	m.registerList.SetSize(halfW, halfH)
	m.globalList.SetSize(halfW, halfH-10)
	m.groupList.SetSize(halfW, halfH-10)

	renderPanel := func(p Panel, l list.Model, header string) string {
		style := inactiveStyle
		if m.active == p {
			style = activeStyle
		}

		// Create a custom header area: List Title + Column Header
		titleStr := boldStyle.PaddingLeft(2).Render(l.Title)
		colHeaderStr := headerStyle.Render(header)

		content := lipgloss.JoinVertical(lipgloss.Left, titleStr, colHeaderStr, l.View())

		return style.Width(halfW).Height(halfH - 10).Render(content)
	}

	projectHeader := fmt.Sprintf(SkillHeaderFormat, "Name", "Register", "Agent", "Source")
	registerHeader := fmt.Sprintf(RegisterHeaderFormat, "Name", "skill_name")

	topL := renderPanel(ProjectPanel, m.projectList, projectHeader)
	topR := renderPanel(RegisterPanel, m.registerList, registerHeader)
	botL := renderPanel(GlobalPanel, m.globalList, projectHeader)
	botR := renderPanel(GroupPanel, m.groupList, registerHeader)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, topL, topR)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, botL, botR)

	dashboard := lipgloss.JoinVertical(lipgloss.Bottom, topRow, bottomRow)

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).PaddingLeft(2).Render(
		"tab: switch panel • a: add • d: remove • enter: detail • q: quit",
	)

	content := lipgloss.JoinVertical(lipgloss.Left, dashboard, help)

	if m.statusMsg != "" {
		status := greenStyle.Bold(true).Padding(0, 1).Render(m.statusMsg)
		return lipgloss.JoinVertical(lipgloss.Left, content, status)
	}

	return content
}

func (m Model) formView() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n  %s\n\n", boldStyle.Render(m.formTitle)))

	for i := range m.formInputs {
		b.WriteString("  " + m.formInputs[i].View() + "\n")
	}

	b.WriteString("\n  (Enter to next/submit, Esc to cancel)\n")
	if m.err != nil {
		b.WriteString(fmt.Sprintf("\n  Error: %v\n", m.err))
	}

	return b.String()
}

func (m Model) detailView() string {
	var title string
	if m.selectedSkill != nil {
		title = boldStyle.Render("Skill Details: " + m.selectedSkill.Name)
	} else if m.selectedGroup != nil {
		title = boldStyle.Render("Group Details: " + m.selectedGroup.Name)
	} else {
		title = boldStyle.Render("Details")
	}

	// Operation box style
	opStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("2")).
		Padding(0, 1).
		MarginTop(1)

	ops := lipgloss.JoinHorizontal(lipgloss.Top,
		greenStyle.Render("[P]"), " Add to Project  ",
		greenStyle.Render("[G]"), " Add to Global   ",
		greenStyle.Render("[U]"), " Add to Group    ",
		greenStyle.Render("[Esc]"), " Back",
	)

	opBox := opStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			boldStyle.Render("Operations:"),
			ops,
		),
	)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.detailViewport.View(),
		opBox,
	)

	if m.err != nil {
		errorMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(fmt.Sprintf("Error: %v", m.err))
		content = lipgloss.JoinVertical(lipgloss.Left, content, errorMsg)
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

func (m Model) groupSelectionView() string {
	m.groupSelectionList.SetSize(m.width-4, m.height-6)

	title := boldStyle.Render("Select Group for " + m.selectedSkill.Name)
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.groupSelectionList.View(),
		"\n (Enter to confirm, Esc to cancel)",
	)

	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

func (m Model) confirmView() string {
	var b strings.Builder
	b.WriteString("\n  " + boldStyle.Render("Confirmation") + "\n\n")
	b.WriteString("  " + m.confirmMsg + "\n\n")
	b.WriteString("  " + greenStyle.Render("[Y] Yes") + "    " + lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("[N] No") + "\n")

	return b.String()
}
