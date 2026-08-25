package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

func TestScanSkills(t *testing.T) {
	app, _ := setupApp(t)

	skillDir := t.TempDir()
	skillSubDir := filepath.Join(skillDir, "my-skill")
	os.MkdirAll(skillSubDir, 0755)
	os.WriteFile(filepath.Join(skillSubDir, "SKILL.md"), []byte("# my-skill\nA test skill"), 0644)

	count, err := app.ScanSkills(skillDir, "")
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	skills, _ := app.ListSkills()
	assert.Len(t, skills, 1)
	assert.Equal(t, "my-skill", skills[0].Name)
}

func TestScanSkillsInvalidDir(t *testing.T) {
	app, _ := setupApp(t)
	_, err := app.ScanSkills("/nonexistent/path/xyz", "")
	assert.Error(t, err)
}

func TestScanSkills_UpdateWithDeletedSkills(t *testing.T) {
	app, configDir := setupApp(t)
	mgr := store.NewManager(configDir)

	repoDir := t.TempDir()
	skill1Dir := filepath.Join(repoDir, "skill-a")
	skill2Dir := filepath.Join(repoDir, "skill-b")
	os.MkdirAll(skill1Dir, 0755)
	os.MkdirAll(skill2Dir, 0755)
	os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte("# skill-a\nSkill A"), 0644)
	os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte("# skill-b\nSkill B"), 0644)

	// Register in registries.json
	regCfg := internal.RegistriesConfig{
		Version: internal.CONFIG_VERSION,
		Registries: []internal.Registry{
			{Name: "org_repo", LocalPath: repoDir},
		},
	}
	assert.NoError(t, mgr.SaveRegistriesConfig(regCfg))

	// Initial scan: 2 skills
	count, err := app.ScanSkills(repoDir, "org_repo")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	skills, err := app.ListSkills()
	assert.NoError(t, err)
	assert.Len(t, skills, 2)

	// Simulate deletion of skill-b
	os.RemoveAll(skill2Dir)

	// Rescan
	count, err = app.ScanSkills(repoDir, "org_repo")
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	skills, err = app.ListSkills()
	assert.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "skill-a", skills[0].Name)

	// Simulate deletion of skill-a (all skills deleted)
	os.RemoveAll(skill1Dir)

	count, err = app.ScanSkills(repoDir, "org_repo")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	skills, err = app.ListSkills()
	assert.NoError(t, err)
	assert.Empty(t, skills)
}

func TestScanSkills_MultiRegistryIsolation(t *testing.T) {
	app, configDir := setupApp(t)
	mgr := store.NewManager(configDir)

	repo1Dir := t.TempDir()
	repo2Dir := t.TempDir()

	s1Dir := filepath.Join(repo1Dir, "s1")
	s2Dir := filepath.Join(repo2Dir, "s2")
	os.MkdirAll(s1Dir, 0755)
	os.MkdirAll(s2Dir, 0755)
	os.WriteFile(filepath.Join(s1Dir, "SKILL.md"), []byte("# s1\nSkill 1"), 0644)
	os.WriteFile(filepath.Join(s2Dir, "SKILL.md"), []byte("# s2\nSkill 2"), 0644)

	regCfg := internal.RegistriesConfig{
		Version: internal.CONFIG_VERSION,
		Registries: []internal.Registry{
			{Name: "repo1", LocalPath: repo1Dir},
			{Name: "repo2", LocalPath: repo2Dir},
		},
	}
	assert.NoError(t, mgr.SaveRegistriesConfig(regCfg))

	_, err := app.ScanSkills(repo1Dir, "repo1")
	assert.NoError(t, err)
	_, err = app.ScanSkills(repo2Dir, "repo2")
	assert.NoError(t, err)

	skills, _ := app.ListSkills()
	assert.Len(t, skills, 2)

	// Delete s1 in repo1, rescan repo1
	os.RemoveAll(s1Dir)
	_, err = app.ScanSkills(repo1Dir, "repo1")
	assert.NoError(t, err)

	// repo2's s2 should still exist
	skills, _ = app.ListSkills()
	assert.Len(t, skills, 1)
	assert.Equal(t, "s2", skills[0].Name)
	assert.Equal(t, "repo2", skills[0].RegistryName)
}
