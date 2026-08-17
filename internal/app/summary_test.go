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

func TestSummary_Empty(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".skill_manager")
	app := New(configDir)

	report, err := app.GetSummaryReport()
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Empty(t, report.Items)
}

func TestSummary_MultiProject(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".skill_manager")
	app := New(configDir)
	mgr := store.NewManager(configDir)
	require.NoError(t, mgr.Init())

	// Save base skills
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

	// 1. Global context: skills [golang, docker], groups [backend, devops]
	globalPath := filepath.Join(configDir, internal.FILENAME_GLOBAL_CONTEXT)
	globalCtx := internal.ContextConfig{
		Version: internal.CONFIG_VERSION,
		Scope:   "global",
		ActiveSkills: []internal.SkillUsage{
			{ID: "s4", Agents: []string{"gemini"}},
			{ID: "s1", Agents: []string{"claude"}},
		},
		ActiveGroups: []internal.GroupUsage{
			{Name: "devops"},
			{Name: "backend"},
		},
	}
	require.NoError(t, mgr.SaveContextConfig(globalCtx, internal.FILENAME_GLOBAL_CONTEXT))

	// 2. Proj1: custom scope "custom-proj", active_skills [python], active_groups [backend]
	proj1Dir := filepath.Join(tempDir, "proj1_folder")
	require.NoError(t, os.MkdirAll(proj1Dir, 0755))
	proj1Path := filepath.Join(proj1Dir, internal.FILENAME_CONTEXT)
	proj1Ctx := internal.ContextConfig{
		Version: internal.CONFIG_VERSION,
		Scope:   "custom-proj",
		ActiveSkills: []internal.SkillUsage{
			{ID: "s2", Agents: []string{"gemini"}},
			{ID: "s2", Agents: []string{"claude"}}, // duplicate skill ID across agents
		},
		ActiveGroups: []internal.GroupUsage{
			{Name: "backend"},
			{Name: "backend"}, // duplicate group
		},
	}
	require.NoError(t, store.SaveLocalContext(proj1Dir, proj1Ctx))

	// 3. Proj2: empty scope -> fallback to folder name "proj2", only groups
	proj2Dir := filepath.Join(tempDir, "proj2")
	require.NoError(t, os.MkdirAll(proj2Dir, 0755))
	proj2Path := filepath.Join(proj2Dir, internal.FILENAME_CONTEXT)
	proj2Ctx := internal.ContextConfig{
		Version:      internal.CONFIG_VERSION,
		Scope:        "",
		ActiveSkills: []internal.SkillUsage{},
		ActiveGroups: []internal.GroupUsage{{Name: "infra"}},
	}
	require.NoError(t, store.SaveLocalContext(proj2Dir, proj2Ctx))

	// 4. Proj3: only skills (including an unknown ID "unknown-skill" which should fallback to ID name)
	proj3Dir := filepath.Join(tempDir, "proj3")
	require.NoError(t, os.MkdirAll(proj3Dir, 0755))
	proj3Path := filepath.Join(proj3Dir, internal.FILENAME_CONTEXT)
	proj3Ctx := internal.ContextConfig{
		Version: internal.CONFIG_VERSION,
		Scope:   "proj3",
		ActiveSkills: []internal.SkillUsage{
			{ID: "s3"},
			{ID: "unknown-skill"},
		},
		ActiveGroups: []internal.GroupUsage{},
	}
	require.NoError(t, store.SaveLocalContext(proj3Dir, proj3Ctx))

	// 5. Context 5: Non-existent context path in summary (should be skipped gracefully)
	missingPath := filepath.Join(tempDir, "missing", internal.FILENAME_CONTEXT)

	// Save Summary
	summary := internal.SummaryConfig{
		Version: internal.CONFIG_VERSION,
		Contexts: []internal.ContextSummary{
			{Dir: "global", ConfigPath: globalPath},
			{Dir: proj1Dir, ConfigPath: proj1Path},
			{Dir: proj2Dir, ConfigPath: proj2Path},
			{Dir: proj3Dir, ConfigPath: proj3Path},
			{Dir: filepath.Dir(missingPath), ConfigPath: missingPath},
		},
	}
	require.NoError(t, mgr.SaveSummaryConfig(summary))

	// Execute GetSummaryReport
	report, err := app.GetSummaryReport()
	require.NoError(t, err)
	require.Len(t, report.Items, 4)

	// Item 0: global
	assert.Equal(t, "global", report.Items[0].ProjectName)
	assert.Equal(t, "global", report.Items[0].Dir)
	assert.Equal(t, []string{"docker", "golang"}, report.Items[0].Skills)
	assert.Equal(t, []string{"backend", "devops"}, report.Items[0].Groups)

	// Item 1: custom-proj
	assert.Equal(t, "custom-proj", report.Items[1].ProjectName)
	assert.Equal(t, proj1Dir, report.Items[1].Dir)
	assert.Equal(t, []string{"python"}, report.Items[1].Skills)
	assert.Equal(t, []string{"backend"}, report.Items[1].Groups)

	// Item 2: proj2
	assert.Equal(t, "proj2", report.Items[2].ProjectName)
	assert.Equal(t, proj2Dir, report.Items[2].Dir)
	assert.Empty(t, report.Items[2].Skills)
	assert.Equal(t, []string{"infra"}, report.Items[2].Groups)

	// Item 3: proj3
	assert.Equal(t, "proj3", report.Items[3].ProjectName)
	assert.Equal(t, proj3Dir, report.Items[3].Dir)
	assert.Equal(t, []string{"rust", "unknown-skill"}, report.Items[3].Skills)
	assert.Empty(t, report.Items[3].Groups)
}
