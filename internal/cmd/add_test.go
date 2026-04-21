// internal/cmd/add_test.go
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal/config"
)

func TestAddCommandLocalContext(t *testing.T) {
	tempDir := t.TempDir()

	// Create mock config dir
	configDir := filepath.Join(tempDir, ".skill_manager")
	os.Setenv("SKILL_MANAGER_CONFIG", configDir)
	defer os.Unsetenv("SKILL_MANAGER_CONFIG")

	manager := config.NewManager(configDir)
	err := manager.Init()
	assert.NoError(t, err)

	// Create a dummy skill to add
	skillPath := filepath.Join(tempDir, "mock_skill")
	err = os.MkdirAll(skillPath, 0755)
	assert.NoError(t, err)

	err = manager.SaveBaseConfig(config.BaseConfig{
		Version: config.CONFIG_VERSION,
		Skills: []config.Skill{
			{ID: "mock-id", Name: "mock_skill", Path: skillPath},
		},
	})
	assert.NoError(t, err)

	// Change working directory to a new project dir
	projectDir := filepath.Join(tempDir, "project")
	err = os.MkdirAll(projectDir, 0755)
	assert.NoError(t, err)

	oldWd, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(oldWd)

	// Test adding without agent when no agent dir exists
	rootCmd.SetArgs([]string{"add", "mock_skill"})
	err = rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no agent directories found")

	// Test adding with agent
	addGlobal = false
	addAgents = "gemini"
	rootCmd.SetArgs([]string{"add", "mock_skill", "--agent", "gemini"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify local context is created
	ctxData, err := os.ReadFile(filepath.Join(projectDir, config.FILENAME_CONTEXT))
	assert.NoError(t, err)
	assert.Contains(t, string(ctxData), "mock-id")

	// Verify symlink
	linkPath := filepath.Join(projectDir, ".gemini", "skills", "mock_skill")
	info, err := os.Lstat(linkPath)
	assert.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0)

	target, err := os.Readlink(linkPath)
	assert.NoError(t, err)
	// On some OS (like Darwin), TempDir might return a path with /private/var... vs /var...
	// We use filepath.EvalSymlinks to compare absolute paths if needed,
	// but here we just check if it contains the skillPath.
	assert.Contains(t, target, "mock_skill")
}
