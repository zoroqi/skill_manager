package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

func TestGetStatus(t *testing.T) {
	app, _ := setupApp(t)
	workDir := t.TempDir()

	_, skills, groups, err := app.GetStatus(workDir, false)
	assert.NoError(t, err)
	assert.Empty(t, skills)
	assert.Empty(t, groups)
}

func TestSyncContext(t *testing.T) {
	app, _ := setupApp(t)
	workDir := t.TempDir()

	err := app.SyncContext(workDir, false, false)
	assert.NoError(t, err)
}

func TestSyncContext_PrunesInvalidActiveSkillsAndGroups(t *testing.T) {
	app, _ := setupApp(t)
	workDir := t.TempDir()

	// 1. Create a valid skill
	s1Dir := filepath.Join(workDir, "s1")
	os.MkdirAll(s1Dir, 0755)
	os.WriteFile(filepath.Join(s1Dir, "SKILL.md"), []byte("# s1"), 0644)

	baseCfg := internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{ID: "s1_id", Name: "s1", Path: s1Dir},
		},
	}
	assert.NoError(t, app.store.SaveBaseConfig(baseCfg))

	// 2. Create a valid group
	groupsCfg := internal.GroupsConfig{
		Version: internal.CONFIG_VERSION,
		Groups: []internal.Group{
			{Name: "valid_group", SkillIDs: []string{"s1_id"}},
		},
	}
	assert.NoError(t, app.store.SaveGroupsConfig(groupsCfg))

	// 3. Create context config with both valid and invalid active skills and groups
	localCtx := internal.ContextConfig{
		Version: internal.CONFIG_VERSION,
		Scope:   "test-project",
		ActiveSkills: []internal.SkillUsage{
			{ID: "s1_id", Agents: []string{"gemini"}},
			{ID: "invalid_skill_id", Agents: []string{"gemini"}},
		},
		ActiveGroups: []internal.GroupUsage{
			{Name: "valid_group", Agents: []string{"gemini"}},
			{Name: "invalid_group", Agents: []string{"gemini"}},
		},
	}
	assert.NoError(t, store.SaveLocalContext(workDir, localCtx))

	// 4. Run SyncContext
	err := app.SyncContext(workDir, false, false)
	assert.NoError(t, err)

	// 5. Verify local context config has pruned invalid skills and groups
	savedCtx, err := store.EnsureLocalContext(workDir)
	assert.NoError(t, err)
	assert.Len(t, savedCtx.ActiveSkills, 1)
	assert.Equal(t, "s1_id", savedCtx.ActiveSkills[0].ID)

	assert.Len(t, savedCtx.ActiveGroups, 1)
	assert.Equal(t, "valid_group", savedCtx.ActiveGroups[0].Name)
}

