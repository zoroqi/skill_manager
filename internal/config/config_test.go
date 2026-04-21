// internal/config/config_test.go
package config

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigJSONFormat(t *testing.T) {
	// 1. Base Skills Config
	skill := Skill{
		ID:          "e4d909c290d0fb1ca068ffaddf22cbd0",
		Name:        "golang-pro",
		Path:        "/some/path",
		Description: "A skill",
		Metadata: map[string]any{
			"scope":         "universal",
			"allowed-tools": []string{"Read", "Write"},
		},
	}
	baseConfig := BaseConfig{
		Version: CONFIG_VERSION,
		Skills:  []Skill{skill},
	}
	data, err := json.Marshal(baseConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"skills":[{`)

	// 2. Groups Config
	group := Group{
		ID:          "g1",
		Name:        "group1",
		Description: "desc",
		SkillIDs:    []string{"e4d909c290d0fb1ca068ffaddf22cbd0"},
	}
	groupsConfig := GroupsConfig{
		Version: CONFIG_VERSION,
		Groups:  []Group{group},
	}
	data, err = json.Marshal(groupsConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"groups":[{`)

	// 3. Context Config
	contextConfig := ContextConfig{
		Version:   CONFIG_VERSION,
		Scope:     "project",
		TargetDir: "/project/dir",
		ActiveSkills: []SkillUsage{
			{ID: "e4d909c290d0fb1ca068ffaddf22cbd0", Agents: []string{"gemini"}},
		},
		ActiveGroups: []string{"g1"},
	}
	data, err = json.Marshal(contextConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"scope":"project"`)

	// 4. Summary Config
	summaryConfig := SummaryConfig{
		Version: CONFIG_VERSION,
		Contexts: []ContextSummary{
			{Dir: "/project/dir", ConfigPath: "/project/dir/.skill_manager.json", LastUpdated: "2026-04-16T10:00:00Z"},
		},
	}
	data, err = json.Marshal(summaryConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"contexts":[{`)
}
