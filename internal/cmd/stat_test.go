package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/app"
	"github.com/zoroqi/skill_manager/internal/store"
)

func TestStatCommands(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".skill_manager")
	os.Setenv("SKILL_MANAGER_CONFIG", configDir)
	defer os.Unsetenv("SKILL_MANAGER_CONFIG")

	application = app.New(configDir)
	mgr := store.NewManager(configDir)
	require.NoError(t, mgr.Init())

	// Save base skills
	baseCfg := internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{ID: "s1", Name: "golang", Path: filepath.Join(tempDir, "s1")},
			{ID: "s2", Name: "python", Path: filepath.Join(tempDir, "s2")},
		},
	}
	require.NoError(t, mgr.SaveBaseConfig(baseCfg))

	// Save groups
	groupsCfg := internal.GroupsConfig{
		Version: internal.CONFIG_VERSION,
		Groups: []internal.Group{
			{Name: "backend", SkillIDs: []string{"s1", "s2"}},
		},
	}
	require.NoError(t, mgr.SaveGroupsConfig(groupsCfg))

	// Project context
	projDir := filepath.Join(tempDir, "proj")
	require.NoError(t, os.MkdirAll(projDir, 0755))
	projCtxPath := filepath.Join(projDir, internal.FILENAME_CONTEXT)
	projCtx := internal.ContextConfig{
		Version:        internal.CONFIG_VERSION,
		Scope:          "proj",
		ActiveGroups:   []internal.GroupUsage{{Name: "backend"}},
		ActiveSkills:   []internal.SkillUsage{},
		InactiveSkills: []internal.InactiveSkillUsage{},
	}
	require.NoError(t, store.SaveLocalContext(projDir, projCtx))

	// Global context
	globalPath := filepath.Join(configDir, internal.FILENAME_GLOBAL_CONTEXT)
	globalCtx := internal.ContextConfig{
		Version:        internal.CONFIG_VERSION,
		Scope:          "global",
		ActiveGroups:   []internal.GroupUsage{{Name: "backend"}},
		ActiveSkills:   []internal.SkillUsage{{ID: "s1"}},
		InactiveSkills: []internal.InactiveSkillUsage{},
	}
	require.NoError(t, mgr.SaveContextConfig(globalCtx, internal.FILENAME_GLOBAL_CONTEXT))

	// Save summary
	summary := internal.SummaryConfig{
		Version: internal.CONFIG_VERSION,
		Contexts: []internal.ContextSummary{
			{Dir: "global", ConfigPath: globalPath},
			{Dir: projDir, ConfigPath: projCtxPath},
		},
	}
	require.NoError(t, mgr.SaveSummaryConfig(summary))

	// 1. Test "stat" (both tables)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"stat"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Skill Name")
	assert.Contains(t, out, "golang")
	assert.Contains(t, out, "python")
	assert.Contains(t, out, "Group Name")
	assert.Contains(t, out, "backend")

	// 2. Test "stat skill"
	buf.Reset()
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"stat", "skill"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	out = buf.String()
	assert.Contains(t, out, "Skill Name")
	assert.Contains(t, out, "golang")
	assert.NotContains(t, out, "Group Name")

	// 3. Test "stat group"
	buf.Reset()
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"stat", "group"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	out = buf.String()
	assert.Contains(t, out, "Group Name")
	assert.Contains(t, out, "backend")
	assert.NotContains(t, out, "Skill Name")

	// 4. Test aliases "stats skills" and "stats groups"
	buf.Reset()
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"stats", "skills"})
	err = rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Skill Name")

	buf.Reset()
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"stats", "groups"})
	err = rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Group Name")
}
