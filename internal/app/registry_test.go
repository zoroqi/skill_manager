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

func TestCleanupSkills_RemovesOrphanedAndMissingSkills(t *testing.T) {
	app, _ := setupApp(t)

	// Create physical dirs
	tempDir := t.TempDir()
	validLocalDir := filepath.Join(tempDir, "valid_local")
	missingLocalDir := filepath.Join(tempDir, "missing_local") // will not have SKILL.md
	repoDir := filepath.Join(tempDir, "repo")
	validRepoSkillDir := filepath.Join(repoDir, "repo_skill")

	os.MkdirAll(validLocalDir, 0755)
	os.WriteFile(filepath.Join(validLocalDir, "SKILL.md"), []byte("# valid-local"), 0644)
	os.MkdirAll(missingLocalDir, 0755)
	// no SKILL.md in missingLocalDir

	os.MkdirAll(validRepoSkillDir, 0755)
	os.WriteFile(filepath.Join(validRepoSkillDir, "SKILL.md"), []byte("# repo-skill"), 0644)

	// Register repo in registries.json
	regCfg := internal.RegistriesConfig{
		Version: internal.CONFIG_VERSION,
		Registries: []internal.Registry{
			{Name: "my_repo", LocalPath: repoDir},
		},
	}
	assert.NoError(t, app.store.SaveRegistriesConfig(regCfg))

	// Save BaseConfig with 4 skills:
	// s1: valid local
	// s2: invalid local (no SKILL.md)
	// s3: valid repo skill
	// s4: invalid repo skill (registry "nonexistent_repo" not in registries.json)
	baseCfg := internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{ID: "s1", Name: "valid-local", Path: validLocalDir},
			{ID: "s2", Name: "missing-local", Path: missingLocalDir},
			{ID: "s3", Name: "repo-skill", RegistryName: "my_repo", Path: "repo_skill"},
			{ID: "s4", Name: "orphan-repo-skill", RegistryName: "nonexistent_repo", Path: "orphan"},
		},
	}
	assert.NoError(t, app.store.SaveBaseConfig(baseCfg))

	// Save GroupsConfig with g1 containing all 4 IDs
	groupsCfg := internal.GroupsConfig{
		Version: internal.CONFIG_VERSION,
		Groups: []internal.Group{
			{Name: "g1", Description: "test group", SkillIDs: []string{"s1", "s2", "s3", "s4"}},
		},
	}
	assert.NoError(t, app.store.SaveGroupsConfig(groupsCfg))

	// Run cleanupSkills
	err := app.cleanupSkills()
	assert.NoError(t, err)

	// Check skills in BaseConfig
	skills, err := app.ListSkills()
	assert.NoError(t, err)
	assert.Len(t, skills, 2)
	skillNames := []string{skills[0].Name, skills[1].Name}
	assert.Contains(t, skillNames, "valid-local")
	assert.Contains(t, skillNames, "repo-skill")

	// Check groups in GroupsConfig
	loadedGroups, err := app.store.LoadGroupsConfig()
	assert.NoError(t, err)
	assert.Len(t, loadedGroups.Groups, 1)
	assert.ElementsMatch(t, []string{"s1", "s3"}, loadedGroups.Groups[0].SkillIDs)
}

func TestRegisterRemove_CleansUpSkillsAndGroups(t *testing.T) {
	app, _ := setupApp(t)

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	skillDir := filepath.Join(repoDir, "skill-x")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# skill-x"), 0644)

	regCfg := internal.RegistriesConfig{
		Version: internal.CONFIG_VERSION,
		Registries: []internal.Registry{
			{Name: "repo_x", LocalPath: repoDir},
		},
	}
	assert.NoError(t, app.store.SaveRegistriesConfig(regCfg))

	_, err := app.ScanSkills(repoDir, "repo_x")
	assert.NoError(t, err)

	skills, err := app.ListSkills()
	assert.NoError(t, err)
	assert.Len(t, skills, 1)
	skillID := skills[0].ID

	// Add to group
	assert.NoError(t, app.store.CreateGroup("dev-group", "desc"))
	assert.NoError(t, app.store.AddSkillToGroup("dev-group", skillID))

	// Remove registry
	err = app.RegisterRemove("repo_x", false)
	assert.NoError(t, err)

	// Check skills are gone
	skills, err = app.ListSkills()
	assert.NoError(t, err)
	assert.Empty(t, skills)

	// Check group has no dangling skill_id
	groups, err := app.store.LoadGroupsConfig()
	assert.NoError(t, err)
	assert.Len(t, groups.Groups, 1)
	assert.Empty(t, groups.Groups[0].SkillIDs)
}
