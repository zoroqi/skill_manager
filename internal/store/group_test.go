package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zoroqi/skill_manager/internal"
)

func TestGroupManagement(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "skill-manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	err = manager.Init()
	require.NoError(t, err)

	err = manager.CreateGroup("test-group", "A test group")
	assert.NoError(t, err)

	group, err := manager.GetGroup("test-group")
	assert.NoError(t, err)
	assert.NotNil(t, group)
	assert.Equal(t, "test-group", group.Name)
	assert.Equal(t, "A test group", group.Description)

	err = manager.AddSkillToGroup("test-group", "skill-1")
	assert.NoError(t, err)
	group, _ = manager.GetGroup("test-group")
	assert.Contains(t, group.SkillIDs, "skill-1")

	err = manager.RemoveSkillFromGroup("test-group", "skill-1")
	assert.NoError(t, err)
	group, _ = manager.GetGroup("test-group")
	assert.NotContains(t, group.SkillIDs, "skill-1")

	err = manager.DeleteGroup("test-group")
	assert.NoError(t, err)
	group, _ = manager.GetGroup("test-group")
	assert.Nil(t, group)
}

func TestSyncContext(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "skill-manager-sync-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configDir := filepath.Join(tempDir, "config")
	projectDir := filepath.Join(tempDir, "project")
	os.MkdirAll(configDir, 0755)
	os.MkdirAll(projectDir, 0755)

	manager := NewManager(configDir)
	manager.Init()

	skill1Path := filepath.Join(tempDir, "skill1")
	os.WriteFile(skill1Path, []byte("skill1"), 0644)
	skill2Path := filepath.Join(tempDir, "skill2")
	os.WriteFile(skill2Path, []byte("skill2"), 0644)

	baseCfg := internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{ID: "s1", Name: "skill-1", Path: skill1Path},
			{ID: "s2", Name: "skill-2", Path: skill2Path},
		},
	}
	manager.SaveBaseConfig(baseCfg)

	manager.CreateGroup("g1", "Group 1")
	manager.AddSkillToGroup("g1", "s1")

	agentDir := filepath.Join(projectDir, ".gemini")
	os.MkdirAll(filepath.Join(agentDir, "skills"), 0755)

	ctxCfg := &internal.ContextConfig{
		ActiveGroups: []string{"g1"},
		ActiveSkills: []internal.SkillUsage{
			{ID: "s2", Agents: []string{"gemini"}},
		},
	}

	err = manager.SyncContext(projectDir, ctxCfg, false)
	assert.NoError(t, err)

	_, err = os.Lstat(filepath.Join(projectDir, ".gemini", "skills", "skill-1"))
	assert.NoError(t, err, "skill-1 symlink should exist")
	_, err = os.Lstat(filepath.Join(projectDir, ".gemini", "skills", "skill-2"))
	assert.NoError(t, err, "skill-2 symlink should exist")

	conflictPath := filepath.Join(projectDir, ".gemini", "skills", "skill-1")
	os.Remove(conflictPath)
	os.WriteFile(conflictPath, []byte("im a file"), 0644)

	err = manager.SyncContext(projectDir, ctxCfg, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conflict")

	err = manager.SyncContext(projectDir, ctxCfg, true)
	assert.NoError(t, err)
	info, _ := os.Lstat(conflictPath)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "Should be a symlink now")

	ctxCfg.ActiveGroups = []string{}
	err = manager.SyncContext(projectDir, ctxCfg, true)
	assert.NoError(t, err)
	_, err = os.Lstat(filepath.Join(projectDir, ".gemini", "skills", "skill-1"))
	assert.Error(t, err, "skill-1 symlink should be removed")
	_, err = os.Lstat(filepath.Join(projectDir, ".gemini", "skills", "skill-2"))
	assert.NoError(t, err, "skill-2 symlink should still exist")
}
