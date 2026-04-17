// internal/cmd/remove_stop_status_test.go
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal/config"
)

func TestRemoveStopStatus(t *testing.T) {
	tempDir := t.TempDir()
	
	// Setup config dir
	configDir := filepath.Join(tempDir, ".skill_manager")
	os.Setenv("SKILL_MANAGER_CONFIG", configDir)
	defer os.Unsetenv("SKILL_MANAGER_CONFIG")
	
	manager := config.NewManager(configDir)
	err := manager.Init()
	assert.NoError(t, err)

	// Create a dummy skill
	skillPath := filepath.Join(tempDir, "my_skill")
	err = os.MkdirAll(skillPath, 0755)
	assert.NoError(t, err)

	err = manager.SaveBaseConfig(config.BaseConfig{
		Version: "1.0",
		Skills: []config.Skill{
			{ID: "skill-1", Name: "my_skill", Path: skillPath},
		},
	})
	assert.NoError(t, err)

	// Project setup
	projectDir := filepath.Join(tempDir, "project")
	err = os.MkdirAll(projectDir, 0755)
	assert.NoError(t, err)
	
	oldWd, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(oldWd)

	// 1. Add skill
	addAgents = "gemini"
	addGlobal = false
	rootCmd.SetArgs([]string{"add", "my_skill", "--agent", "gemini"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify linked
	linkPath := filepath.Join(projectDir, ".gemini", "skills", "my_skill")
	_, err = os.Lstat(linkPath)
	assert.NoError(t, err)

	// 2. Test status (Active)
	statusGlobal = false
	rootCmd.SetArgs([]string{"status"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// 3. Stop skill (partial deactivation)
	// First add another agent to test partial deactivation
	addAgents = "claude"
	rootCmd.SetArgs([]string{"add", "my_skill", "--agent", "claude"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	ctxCfg, err := config.EnsureLocalContext(projectDir)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"gemini", "claude"}, ctxCfg.ActiveSkills[0].Agents)

	// Now stop ONLY gemini
	stopAgents = "gemini"
	stopGlobal = false
	rootCmd.SetArgs([]string{"stop", "my_skill", "--agent", "gemini"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify gemini link removed but claude link remains
	_, err = os.Lstat(linkPath) // gemini link
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	claudeLinkPath := filepath.Join(projectDir, ".claude", "skills", "my_skill")
	_, err = os.Lstat(claudeLinkPath)
	assert.NoError(t, err)

	// Verify skill remains ACTIVE with only claude
	ctxCfg, err = config.EnsureLocalContext(projectDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ctxCfg.ActiveSkills))
	assert.Equal(t, []string{"claude"}, ctxCfg.ActiveSkills[0].Agents)
	assert.Equal(t, 0, len(ctxCfg.InactiveSkills))

	// Now stop ALL (no agent flag)
	stopAgents = ""
	rootCmd.SetArgs([]string{"stop", "my_skill"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify all links removed and skill is INACTIVE
	_, err = os.Lstat(claudeLinkPath)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	ctxCfg, err = config.EnsureLocalContext(projectDir)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ctxCfg.ActiveSkills))
	assert.Equal(t, 1, len(ctxCfg.InactiveSkills))
	assert.Equal(t, "skill-1", ctxCfg.InactiveSkills[0].ID)

	// 4. Test re-activation (Add again)
	addAgents = "claude"
	rootCmd.SetArgs([]string{"add", "my_skill", "--agent", "claude"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	ctxCfg, err = config.EnsureLocalContext(projectDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ctxCfg.ActiveSkills))
	assert.Equal(t, 0, len(ctxCfg.InactiveSkills))
	assert.Equal(t, "skill-1", ctxCfg.ActiveSkills[0].ID)
	// Agents should be just claude because gemini was stopped earlier
	assert.ElementsMatch(t, []string{"claude"}, ctxCfg.ActiveSkills[0].Agents)

	// 5. Test status
	rootCmd.SetArgs([]string{"status"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// 6. Remove skill
	removeGlobal = false
	rootCmd.SetArgs([]string{"remove", "my_skill"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify context cleared
	ctxCfg, err = config.EnsureLocalContext(projectDir)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ctxCfg.ActiveSkills))
	assert.Equal(t, 0, len(ctxCfg.InactiveSkills))
}
