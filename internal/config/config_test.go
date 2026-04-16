// internal/config/config_test.go
package config

import (
	"encoding/json"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestConfigJSONFormat(t *testing.T) {
	// 1. Base Skills Config
	skill := Skill{
		ID:          "e4d909c290d0fb1ca068ffaddf22cbd0",
		Name:        "golang-pro",
		Path:        "/some/path",
		Description: "A skill",
	}
	baseConfig := BaseConfig{
		Version: "1.0",
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
		Version: "1.0",
		Groups:  []Group{group},
	}
	data, err = json.Marshal(groupsConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"groups":[{`)

	// 3. Context Config
	contextConfig := ContextConfig{
		Version:      "1.0",
		Scope:        "project",
		TargetDir:    "/project/dir",
		ActiveSkills: []string{"e4d909c290d0fb1ca068ffaddf22cbd0"},
		ActiveGroups: []string{"g1"},
	}
	data, err = json.Marshal(contextConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"scope":"project"`)

	// 4. Summary Config
	summaryConfig := SummaryConfig{
		Version: "1.0",
		Contexts: []ContextSummary{
			{Dir: "/project/dir", ConfigPath: "/project/dir/.skill_manager.json", LastUpdated: "2026-04-16T10:00:00Z"},
		},
	}
	data, err = json.Marshal(summaryConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"contexts":[{`)
}
