package tui

import "fmt"

// Implement list.Item for ResolvedSkill
func (s ResolvedSkill) Title() string { return s.ID }
func (s ResolvedSkill) Description() string {
	return fmt.Sprintf("Reg: %s | Agent: %s | Source: %s", s.Register, s.Agents, s.Source)
}
func (s ResolvedSkill) FilterValue() string { return s.ID }

// Implement list.Item for RegisterItem
func (r RegisterItem) Title() string { return r.Name }
func (r RegisterItem) Description() string {
	if r.IsRegistry {
		return r.URL
	}
	if r.Skill != nil {
		return "Skill"
	}
	return ""
}
func (r RegisterItem) FilterValue() string { return r.Name }

// Implement list.Item for GroupItem
func (g GroupItem) Title() string { return g.Name }
func (g GroupItem) Description() string {
	if g.IsGroup {
		return g.Summary
	}
	if g.Skill != nil {
		return "Skill"
	}
	return ""
}
func (g GroupItem) FilterValue() string { return g.Name }
