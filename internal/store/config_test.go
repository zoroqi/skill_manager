package store

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
	"testing"
)

func TestConfigJSONFormat(t *testing.T) {
	skill := internal.Skill{
		ID:          "e4d909c290d0fb1ca068ffaddf22cbd0",
		Name:        "golang-pro",
		Path:        "/some/path",
		Description: "A skill",
		Metadata: map[string]any{
			"scope":         "universal",
			"allowed-tools": []string{"Read", "Write"},
		},
	}
	baseConfig := internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills:  []internal.Skill{skill},
	}
	data, err := json.Marshal(baseConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"skills":[{`)

	group := internal.Group{
		ID:          "g1",
		Name:        "group1",
		Description: "desc",
		SkillIDs:    []string{"e4d909c290d0fb1ca068ffaddf22cbd0"},
	}
	groupsConfig := internal.GroupsConfig{
		Version: internal.CONFIG_VERSION,
		Groups:  []internal.Group{group},
	}
	data, err = json.Marshal(groupsConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"groups":[{`)

	contextConfig := internal.ContextConfig{
		Version:   internal.CONFIG_VERSION,
		Scope:     "project",
		TargetDir: "/project/dir",
		ActiveSkills: []internal.SkillUsage{
			{ID: "e4d909c290d0fb1ca068ffaddf22cbd0", Agents: []string{"gemini"}},
		},
		ActiveGroups: []string{"g1"},
	}
	data, err = json.Marshal(contextConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"scope":"project"`)

	summaryConfig := internal.SummaryConfig{
		Version: internal.CONFIG_VERSION,
		Contexts: []internal.ContextSummary{
			{Dir: "/project/dir", ConfigPath: "/project/dir/.skill_manager.json", LastUpdated: "2026-04-16T10:00:00Z"},
		},
	}
	data, err = json.Marshal(summaryConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"contexts":[{`)
}
