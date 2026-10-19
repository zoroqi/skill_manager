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
		Version: internal.CONFIG_VERSION,
		Scope:   "project",
		ActiveSkills: []internal.SkillUsage{
			{ID: "e4d909c290d0fb1ca068ffaddf22cbd0", Agents: []string{"gemini"}},
		},
		ActiveGroups: []internal.GroupUsage{{Name: "g1"}},
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

func TestBaseConfig_DeletedSkills(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewManager(tempDir)
	assert.NoError(t, mgr.Init())

	baseCfg := internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{ID: "s1", Name: "valid-skill", Path: "/path/valid"},
		},
		DeletedSkills: []internal.Skill{
			{ID: "s2", Name: "deleted-skill", Path: "/path/deleted"},
		},
	}
	assert.NoError(t, mgr.SaveBaseConfig(baseCfg))

	loaded, err := mgr.LoadBaseConfig()
	assert.NoError(t, err)
	assert.Len(t, loaded.Skills, 1)
	assert.Equal(t, "valid-skill", loaded.Skills[0].Name)
	assert.Len(t, loaded.DeletedSkills, 1)
	assert.Equal(t, "deleted-skill", loaded.DeletedSkills[0].Name)
}
