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

func TestSummaryCommand(t *testing.T) {
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
			{ID: "s3", Name: "docker", Path: filepath.Join(tempDir, "s3")},
		},
	}
	require.NoError(t, mgr.SaveBaseConfig(baseCfg))

	// Save groups
	groupsCfg := internal.GroupsConfig{
		Version: internal.CONFIG_VERSION,
		Groups: []internal.Group{
			{Name: "backend", SkillIDs: []string{"s1", "s2"}},
			{Name: "devops", SkillIDs: []string{"s3"}},
		},
	}
	require.NoError(t, mgr.SaveGroupsConfig(groupsCfg))

	// Global context: skill "golang", group "backend"
	globalPath := filepath.Join(configDir, internal.FILENAME_GLOBAL_CONTEXT)
	globalCtx := internal.ContextConfig{
		Version:        internal.CONFIG_VERSION,
		Scope:          "global",
		ActiveSkills:   []internal.SkillUsage{{ID: "s1"}},
		ActiveGroups:   []internal.GroupUsage{{Name: "backend"}},
		InactiveSkills: []internal.InactiveSkillUsage{},
	}
	require.NoError(t, mgr.SaveContextConfig(globalCtx, internal.FILENAME_GLOBAL_CONTEXT))

	// Project context: proj1 with skills "docker", "python", groups "backend", "devops"
	proj1Dir := filepath.Join(tempDir, "proj1")
	require.NoError(t, os.MkdirAll(proj1Dir, 0755))
	proj1Path := filepath.Join(proj1Dir, internal.FILENAME_CONTEXT)
	proj1Ctx := internal.ContextConfig{
		Version:        internal.CONFIG_VERSION,
		Scope:          "proj1",
		ActiveSkills:   []internal.SkillUsage{{ID: "s3"}, {ID: "s2"}},
		ActiveGroups:   []internal.GroupUsage{{Name: "backend"}, {Name: "devops"}},
		InactiveSkills: []internal.InactiveSkillUsage{},
	}
	require.NoError(t, store.SaveLocalContext(proj1Dir, proj1Ctx))

	// Save summary config
	summary := internal.SummaryConfig{
		Version: internal.CONFIG_VERSION,
		Contexts: []internal.ContextSummary{
			{Dir: "global", ConfigPath: globalPath},
			{Dir: proj1Dir, ConfigPath: proj1Path},
		},
	}
	require.NoError(t, mgr.SaveSummaryConfig(summary))

	// 1. Test "summary"
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"summary"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Project")
	assert.Contains(t, out, "Skill")
	assert.Contains(t, out, "Group")
	assert.Contains(t, out, "global")
	assert.Contains(t, out, "golang")
	assert.Contains(t, out, "backend")
	assert.Contains(t, out, "proj1")
	assert.Contains(t, out, "docker")
	assert.Contains(t, out, "python")
	assert.Contains(t, out, "devops")

	// 2. Test alias "summaries"
	buf.Reset()
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"summaries"})
	err = rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Project")
	assert.Contains(t, buf.String(), "global")
	assert.Contains(t, buf.String(), "proj1")
}
