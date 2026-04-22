package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApp_ResolveRegisterPath(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "sm_test")
	defer os.RemoveAll(tmpDir)
	a := New(tmpDir)

	// Test case 1: customPath
	path, err := a.resolveRegisterPath("/tmp/custom", "repo")
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/custom", path)

	// Test case 2: env SKILL_MANAGER_REGISTER_PATH
	os.Setenv("SKILL_MANAGER_REGISTER_PATH", "/tmp/env")
	defer os.Unsetenv("SKILL_MANAGER_REGISTER_PATH")
	path, err = a.resolveRegisterPath("", "repo")
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/env/repo", path)

	// Test case 3: Default (cwd/registries/name)
	os.Unsetenv("SKILL_MANAGER_REGISTER_PATH")
	cwd, _ := os.Getwd()
	path, err = a.resolveRegisterPath("", "repo")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(cwd, "registries", "repo"), path)
}

func TestExtractRepoName(t *testing.T) {
	assert.Equal(t, "user_repo", extractRepoName("https://github.com/user/repo.git"))
	assert.Equal(t, "user_repo", extractRepoName("https://github.com/user/repo"))
	assert.Equal(t, "user_repo", extractRepoName("git@github.com:user/repo.git"))
	assert.Equal(t, "github.com_repo", extractRepoName("github.com/repo.git"))
}
