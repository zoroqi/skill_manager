package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
)

func TestCommands(t *testing.T) {
	// Setup a temporary config directory for the test
	tempConfig := t.TempDir()
	os.Setenv("SKILL_MANAGER_CONFIG", tempConfig)
	defer os.Unsetenv("SKILL_MANAGER_CONFIG")

	// 1. Test init
	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.DirExists(t, tempConfig)
	assert.FileExists(t, filepath.Join(tempConfig, internal.FILENAME_SKILLS))

	// 2. Test scan
	mockDir := t.TempDir()
	skillDir := filepath.Join(mockDir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("test"), 0644)

	rootCmd.SetArgs([]string{"scan", mockDir})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// 3. Test list
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"list"})
	err = rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "test-skill")
}
