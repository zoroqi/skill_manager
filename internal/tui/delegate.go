package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("2")).Bold(true)
	parentStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
)

var RegisterHeaderFormat = "%-35s   %-40s"
var SkillHeaderFormat = "%-20s %-20s %-15s %-20s"

type skillDelegate struct{}

func (d skillDelegate) Height() int                               { return 1 }
func (d skillDelegate) Spacing() int                              { return 0 }
func (d skillDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d skillDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	var s string

	switch item := listItem.(type) {
	case ResolvedSkill:
		s = fmt.Sprintf(SkillHeaderFormat,
			truncate(item.ID, 20),
			truncate(item.Register, 20),
			truncate(item.Agents, 15),
			truncate(item.Source, 20))
	case RegisterItem:
		if item.IsRegistry {
			s = parentStyle.Render(fmt.Sprintf(RegisterHeaderFormat,
				truncate(item.Name, 35),
				truncate(item.URL, 40)))
		} else {
			// Plain indentation under skill_name column
			s = fmt.Sprintf(RegisterHeaderFormat, "", truncate(item.Name, 40))
		}
	case GroupItem:
		if item.IsGroup {
			s = parentStyle.Render(fmt.Sprintf(RegisterHeaderFormat,
				truncate(item.Name, 35),
				truncate(item.Summary, 40)))
		} else {
			// Plain indentation under skill_name column
			s = fmt.Sprintf(RegisterHeaderFormat, "", truncate(item.Name, 40))
		}
	}

	if index == m.Index() {
		fmt.Fprintf(w, "%s", selectedItemStyle.Render("> "+s))
	} else {
		fmt.Fprintf(w, "%s", itemStyle.Render("  "+s))
	}
}

func truncate(s string, w int) string {
	if len(s) > w {
		if w > 3 {
			return s[:w-3] + "..."
		}
		return s[:w]
	}
	return s
}
