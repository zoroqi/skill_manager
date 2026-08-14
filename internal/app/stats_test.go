package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

func TestStats_Empty(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".skill_manager")
	app := New(configDir)

	// No skills or groups registered
	report, err := app.GetStatsReport()
	require.NoError(t, err)
	assert.Empty(t, report.Skills)
	assert.Empty(t, report.Groups)

	skillStats, err := app.GetSkillStats()
	require.NoError(t, err)
	assert.Empty(t, skillStats)

	groupStats, err := app.GetGroupStats()
	require.NoError(t, err)
	assert.Empty(t, groupStats)
}

func TestStats_CountAndSorting(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".skill_manager")
	app := New(configDir)
	mgr := store.NewManager(configDir)
	require.NoError(t, mgr.Init())

	// Register skills: golang (s1), python (s2), rust (s3), docker (s4)
	baseCfg := internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{ID: "s1", Name: "golang", Path: filepath.Join(tempDir, "s1")},
			{ID: "s2", Name: "python", Path: filepath.Join(tempDir, "s2")},
			{ID: "s3", Name: "rust", Path: filepath.Join(tempDir, "s3")},
			{ID: "s4", Name: "docker", Path: filepath.Join(tempDir, "s4")},
		},
	}
	require.NoError(t, mgr.SaveBaseConfig(baseCfg))

	// Register groups: backend (s1, s2), devops (s4), unused_group (s3)
	groupsCfg := internal.GroupsConfig{
		Version: internal.CONFIG_VERSION,
		Groups: []internal.Group{
			{Name: "backend", SkillIDs: []string{"s1", "s2"}},
			{Name: "devops", SkillIDs: []string{"s4"}},
			{Name: "unused_group", SkillIDs: []string{"s3"}},
		},
	}
	require.NoError(t, mgr.SaveGroupsConfig(groupsCfg))

	// Setup contexts:
	// Context 1: Global -> active_groups: [backend], active_skills: []
	globalPath := filepath.Join(configDir, internal.FILENAME_GLOBAL_CONTEXT)
	globalCtx := internal.ContextConfig{
		Version:        internal.CONFIG_VERSION,
		Scope:          "global",
		ActiveGroups:   []internal.GroupUsage{{Name: "backend"}},
		ActiveSkills:   []internal.SkillUsage{},
		InactiveSkills: []internal.InactiveSkillUsage{},
	}
	require.NoError(t, mgr.SaveContextConfig(globalCtx, internal.FILENAME_GLOBAL_CONTEXT))

	// Context 2: Project 1 -> active_groups: [backend, devops], active_skills: [s1]
	proj1Dir := filepath.Join(tempDir, "proj1")
	require.NoError(t, os.MkdirAll(proj1Dir, 0755))
	proj1Path := filepath.Join(proj1Dir, internal.FILENAME_CONTEXT)
	proj1Ctx := internal.ContextConfig{
		Version:        internal.CONFIG_VERSION,
		Scope:          "proj1",
		ActiveGroups:   []internal.GroupUsage{{Name: "backend"}, {Name: "devops"}},
		ActiveSkills:   []internal.SkillUsage{{ID: "s1"}},
		InactiveSkills: []internal.InactiveSkillUsage{},
	}
	require.NoError(t, store.SaveLocalContext(proj1Dir, proj1Ctx))

	// Context 3: Project 2 -> active_groups: [backend], inactive_skills: [python], active_skills: [s4]
	proj2Dir := filepath.Join(tempDir, "proj2")
	require.NoError(t, os.MkdirAll(proj2Dir, 0755))
	proj2Path := filepath.Join(proj2Dir, internal.FILENAME_CONTEXT)
	proj2Ctx := internal.ContextConfig{
		Version:      internal.CONFIG_VERSION,
		Scope:        "proj2",
		ActiveGroups: []internal.GroupUsage{{Name: "backend"}},
		ActiveSkills: []internal.SkillUsage{{ID: "s4"}},
		InactiveSkills: []internal.InactiveSkillUsage{
			{Name: "python"}, // python is blocked in proj2
		},
	}
	require.NoError(t, store.SaveLocalContext(proj2Dir, proj2Ctx))

	// Context 4: Non-existent context path in summary (should be skipped gracefully)
	missingPath := filepath.Join(tempDir, "missing", internal.FILENAME_CONTEXT)

	// Save Summary
	summary := internal.SummaryConfig{
		Version: internal.CONFIG_VERSION,
		Contexts: []internal.ContextSummary{
			{Dir: "global", ConfigPath: globalPath},
			{Dir: proj1Dir, ConfigPath: proj1Path},
			{Dir: proj2Dir, ConfigPath: proj2Path},
			{Dir: filepath.Dir(missingPath), ConfigPath: missingPath},
		},
	}
	require.NoError(t, mgr.SaveSummaryConfig(summary))

	// Run stats
	report, err := app.GetStatsReport()
	require.NoError(t, err)

	// Expected Group Stats:
	// backend: used in global, proj1, proj2 => 3
	// devops: used in proj1 => 1
	// unused_group: used in 0 => excluded
	expectedGroups := []internal.StatItem{
		{Name: "backend", Count: 3},
		{Name: "devops", Count: 1},
	}
	assert.Equal(t, expectedGroups, report.Groups)

	// Expected Skill Stats:
	// golang (s1):
	//   - global: via backend => active (+1)
	//   - proj1: via backend & active_skills => active (+1)
	//   - proj2: via backend => active (+1)
	//   => total = 3
	// docker (s4):
	//   - global: inactive
	//   - proj1: via devops => active (+1)
	//   - proj2: via active_skills => active (+1)
	//   => total = 2
	// python (s2):
	//   - global: via backend => active (+1)
	//   - proj1: via backend => active (+1)
	//   - proj2: blocked by InactiveSkills => inactive (0)
	//   => total = 2
	// rust (s3):
	//   - 0 uses => excluded
	// Sorting: Count desc, then Name asc ->
	// golang: 3
	// docker: 2 (d before p)
	// python: 2
	expectedSkills := []internal.StatItem{
		{Name: "golang", Count: 3},
		{Name: "docker", Count: 2},
		{Name: "python", Count: 2},
	}
	assert.Equal(t, expectedSkills, report.Skills)
}
