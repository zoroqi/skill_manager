package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

func TestGroupUsageCLI(t *testing.T) {
	tempDir := t.TempDir()

	// Setup config dir
	configDir := filepath.Join(tempDir, ".skill_manager")
	os.Setenv("SKILL_MANAGER_CONFIG", configDir)
	defer os.Unsetenv("SKILL_MANAGER_CONFIG")

	manager := store.NewManager(configDir)
	err := manager.Init()
	assert.NoError(t, err)

	// Create a dummy skill and group
	skillPath := filepath.Join(tempDir, "skill1")
	os.MkdirAll(skillPath, 0755)

	baseCfg := internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{ID: "s1", Name: "skill1", Path: skillPath},
		},
	}
	manager.SaveBaseConfig(baseCfg)

	groupsCfg := internal.GroupsConfig{
		Version: internal.CONFIG_VERSION,
		Groups: []internal.Group{
			{Name: "g1", SkillIDs: []string{"s1"}},
		},
	}
	manager.SaveGroupsConfig(groupsCfg)

	// Project setup
	projectDir := filepath.Join(tempDir, "project")
	os.MkdirAll(projectDir, 0755)

	// Create agent dirs
	os.MkdirAll(filepath.Join(projectDir, ".gemini", "skills"), 0755)
	os.MkdirAll(filepath.Join(projectDir, ".claude", "skills"), 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(oldWd)

	// Reset globals for test
	addGroup = ""
	addAgents = ""
	addGlobal = false
	removeGroup = ""
	removeAgents = ""
	removeGlobal = false

	// 1. Add group with specific agents
	addGroup = "g1"
	addAgents = "gemini"
	addGlobal = false
	rootCmd.SetArgs([]string{"add", "--group", "g1", "--agent", "gemini"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify context
	ctxCfg, _ := store.EnsureLocalContext(projectDir)
	assert.Len(t, ctxCfg.ActiveGroups, 1)
	assert.Equal(t, "g1", ctxCfg.ActiveGroups[0].Name)
	assert.Equal(t, []string{"gemini"}, ctxCfg.ActiveGroups[0].Agents)

	// Verify symlink
	_, err = os.Lstat(filepath.Join(projectDir, ".gemini", "skills", "skill1"))
	assert.NoError(t, err)
	_, err = os.Lstat(filepath.Join(projectDir, ".claude", "skills", "skill1"))
	assert.Error(t, err) // Should not be in claude

	// 2. Add more agents to the same group
	addAgents = "claude"
	rootCmd.SetArgs([]string{"add", "--group", "g1", "--agent", "claude"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	ctxCfg, _ = store.EnsureLocalContext(projectDir)
	assert.ElementsMatch(t, []string{"gemini", "claude"}, ctxCfg.ActiveGroups[0].Agents)
	_, err = os.Lstat(filepath.Join(projectDir, ".claude", "skills", "skill1"))
	assert.NoError(t, err)

	// 3. Remove one agent from the group
	removeGroup = "g1"
	removeAgents = "gemini"
	removeGlobal = false
	rootCmd.SetArgs([]string{"remove", "--group", "g1", "--agent", "gemini"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	ctxCfg, _ = store.EnsureLocalContext(projectDir)
	assert.Equal(t, []string{"claude"}, ctxCfg.ActiveGroups[0].Agents)
	_, err = os.Lstat(filepath.Join(projectDir, ".gemini", "skills", "skill1"))
	assert.Error(t, err) // gemini link should be gone
	_, err = os.Lstat(filepath.Join(projectDir, ".claude", "skills", "skill1"))
	assert.NoError(t, err) // claude link should stay

	// 4. Remove the group entirely
	removeAgents = ""
	rootCmd.SetArgs([]string{"remove", "--group", "g1"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	ctxCfg, _ = store.EnsureLocalContext(projectDir)
	assert.Empty(t, ctxCfg.ActiveGroups)
	_, err = os.Lstat(filepath.Join(projectDir, ".claude", "skills", "skill1"))
	assert.Error(t, err) // claude link should be gone
}
