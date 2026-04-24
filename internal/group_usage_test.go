package internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupUsage_UnmarshalJSON(t *testing.T) {
	t.Run("unmarshal from string (backward compatibility)", func(t *testing.T) {
		var gu GroupUsage
		data := []byte(`"group1"`)
		err := json.Unmarshal(data, &gu)
		assert.NoError(t, err)
		assert.Equal(t, "group1", gu.Name)
		assert.Empty(t, gu.Agents)
	})

	t.Run("unmarshal from object", func(t *testing.T) {
		var gu GroupUsage
		data := []byte(`{"name":"group2","agents":["gemini","claude"]}`)
		err := json.Unmarshal(data, &gu)
		assert.NoError(t, err)
		assert.Equal(t, "group2", gu.Name)
		assert.Equal(t, []string{"gemini", "claude"}, gu.Agents)
	})

	t.Run("unmarshal ContextConfig with mixed types (though unlikely)", func(t *testing.T) {
		var cc ContextConfig
		data := []byte(`{
			"version": "1.0",
			"scope": "project",
			"active_skills": [],
			"active_groups": ["old_group", {"name": "new_group", "agents": ["gemini"]}]
		}`)
		err := json.Unmarshal(data, &cc)
		assert.NoError(t, err)
		assert.Len(t, cc.ActiveGroups, 2)
		assert.Equal(t, "old_group", cc.ActiveGroups[0].Name)
		assert.Empty(t, cc.ActiveGroups[0].Agents)
		assert.Equal(t, "new_group", cc.ActiveGroups[1].Name)
		assert.Equal(t, []string{"gemini"}, cc.ActiveGroups[1].Agents)
	})
	t.Run("marshal to object", func(t *testing.T) {
		gu := GroupUsage{Name: "group3", Agents: []string{"gemini"}}
		data, err := json.Marshal(gu)
		assert.NoError(t, err)
		assert.Equal(t, `{"name":"group3","agents":["gemini"]}`, string(data))
	})
}
