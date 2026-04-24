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

func TestPathRelativization(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir)

	// Setup registry
	reg := internal.Registry{
		Name:      "test_reg",
		LocalPath: filepath.Join(tempDir, "registries", "test_reg"),
	}
	err := manager.SaveRegistriesConfig(internal.RegistriesConfig{
		Version:    internal.CONFIG_VERSION,
		Registries: []internal.Registry{reg},
	})
	assert.NoError(t, err)

	physicalPath, err := reg.GetPhysicalPath()
	assert.NoError(t, err)

	// Save skill with absolute path
	skillPath := filepath.Join(physicalPath, "skill_a")
	cfg := internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{
				ID:           "skill_a",
				Name:         "skill_a",
				RegistryName: "test_reg",
				Path:         skillPath,
			},
		},
	}
	err = manager.SaveBaseConfig(cfg)
	assert.NoError(t, err)

	// Load and verify absolute path in memory
	cfgLoaded, err := manager.LoadBaseConfig()
	assert.NoError(t, err)
	assert.Equal(t, skillPath, cfgLoaded.Skills[0].Path)

	// Verify relative path on disk
	var diskCfg internal.BaseConfig
	err = manager.loadJSON(internal.FILENAME_SKILLS, &diskCfg)
	assert.NoError(t, err)
	assert.Equal(t, "skill_a", diskCfg.Skills[0].Path)
}

func TestUpdateContextSummary(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir)

	err := manager.Init()
	assert.NoError(t, err)

	dir := "/path/to/project"
	configPath := "/path/to/project/context.json"
	err = manager.UpdateContextSummary(dir, configPath)
	assert.NoError(t, err)

	summary, err := manager.LoadSummaryConfig()
	assert.NoError(t, err)

	// Should have global and the new one
	assert.Len(t, summary.Contexts, 2)

	found := false
	for _, ctx := range summary.Contexts {
		if ctx.Dir == dir {
			assert.Equal(t, configPath, ctx.ConfigPath)
			assert.NotEmpty(t, ctx.LastUpdated)
			found = true
			break
		}
	}
	assert.True(t, found)

	// Update again
	err = manager.UpdateContextSummary(dir, configPath)
	assert.NoError(t, err)
	summary2, _ := manager.LoadSummaryConfig()
	assert.Len(t, summary2.Contexts, 2)
}
