package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
)

func TestApp_ResolveRegisterPath(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "sm_test")
	defer os.RemoveAll(tmpDir)
	a := New(tmpDir)

	// Test case 1: customPath
	logical, physical, err := a.resolveRegisterPath("/tmp/custom", "repo")
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/custom", logical)
	assert.Equal(t, "/tmp/custom", physical)

	// Test case 2: env SKILL_MANAGER_REGISTER_PATH
	os.Setenv(internal.ENV_REGISTER_PATH, "/tmp/env")
	defer os.Unsetenv(internal.ENV_REGISTER_PATH)
	logical, physical, err = a.resolveRegisterPath("", "repo")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("$"+internal.ENV_REGISTER_PATH, "repo"), logical)
	assert.Equal(t, "/tmp/env/repo", physical)

	// Test case 3: Default (cwd/registries/name)
	os.Unsetenv(internal.ENV_REGISTER_PATH)
	cwd, _ := os.Getwd()
	logical, physical, err = a.resolveRegisterPath("", "repo")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("$"+internal.ENV_REGISTER_PATH, "repo"), logical)
	assert.Equal(t, filepath.Join(cwd, "registries", "repo"), physical)
}

func TestExtractRepoName(t *testing.T) {
	assert.Equal(t, "user_repo", extractRepoName("https://github.com/user/repo.git"))
	assert.Equal(t, "user_repo", extractRepoName("https://github.com/user/repo"))
	assert.Equal(t, "user_repo", extractRepoName("git@github.com:user/repo.git"))
	assert.Equal(t, "github.com_repo", extractRepoName("github.com/repo.git"))
}
