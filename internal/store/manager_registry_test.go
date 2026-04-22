package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
)

func TestManager_Registry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skill_manager_test_registry")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	err = m.Init()
	assert.NoError(t, err)

	// Test Initial Load
	cfg, err := m.LoadRegistriesConfig()
	assert.NoError(t, err)
	assert.Equal(t, internal.CONFIG_VERSION, cfg.Version)
	assert.Empty(t, cfg.Registries)

	// Test Save and Load
	newReg := internal.Registry{
		Name:        "test-repo",
		URL:         "https://github.com/user/test-repo",
		LocalPath:   "/tmp/test-repo",
		LastUpdated: "2023-01-01T00:00:00Z",
	}
	cfg.Registries = append(cfg.Registries, newReg)
	err = m.SaveRegistriesConfig(cfg)
	assert.NoError(t, err)

	loadedCfg, err := m.LoadRegistriesConfig()
	assert.NoError(t, err)
	assert.Len(t, loadedCfg.Registries, 1)
	assert.Equal(t, "test-repo", loadedCfg.Registries[0].Name)

	// Verify file exists
	assert.FileExists(t, filepath.Join(tmpDir, internal.FILENAME_REGISTRIES))
}
