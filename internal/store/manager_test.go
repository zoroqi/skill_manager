package store

import (
	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadBaseConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir)

	err := manager.Init()
	assert.NoError(t, err)

	filePath := filepath.Join(tempDir, internal.FILENAME_SKILLS)
	assert.FileExists(t, filePath)

	cfg, err := manager.LoadBaseConfig()
	assert.NoError(t, err)
	assert.Equal(t, internal.CONFIG_VERSION, cfg.Version)
	assert.Empty(t, cfg.Skills)

	cfg.Skills = append(cfg.Skills, internal.Skill{ID: "123", Name: "test"})
	err = manager.SaveBaseConfig(cfg)
	assert.NoError(t, err)

	cfg2, err := manager.LoadBaseConfig()
	assert.NoError(t, err)
	assert.Len(t, cfg2.Skills, 1)
	assert.Equal(t, "123", cfg2.Skills[0].ID)
}
