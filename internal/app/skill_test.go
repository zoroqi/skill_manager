package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

func setupApp(t *testing.T) (*App, string) {
	t.Helper()
	configDir := t.TempDir()
	app := New(configDir)
	require.NoError(t, app.Initialize())
	return app, configDir
}

func TestListSkills(t *testing.T) {
	app, configDir := setupApp(t)
	_ = configDir

	skills, err := app.ListSkills()
	assert.NoError(t, err)
	assert.Empty(t, skills)
}

func TestAddSkill(t *testing.T) {
	app, configDir := setupApp(t)

	skillPath := filepath.Join(configDir, "my-skill")
	os.WriteFile(skillPath, []byte("skill"), 0644)

	mgr := store.NewManager(configDir)
	cfg, _ := mgr.LoadBaseConfig()
	cfg.Skills = append(cfg.Skills, internal.Skill{ID: "s1", Name: "my-skill", Path: skillPath})
	mgr.SaveBaseConfig(cfg)

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".gemini", "skills"), 0755)

	err := app.AddSkill(projectDir, "my-skill", "", []string{"gemini"}, false, false)
	assert.NoError(t, err)

	_, err = os.Lstat(filepath.Join(projectDir, ".gemini", "skills", "my-skill"))
	assert.NoError(t, err)
}

func TestAddSkillNotFound(t *testing.T) {
	app, _ := setupApp(t)
	err := app.AddSkill(t.TempDir(), "nonexistent", "", []string{"gemini"}, false, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoveSkill(t *testing.T) {
	app, configDir := setupApp(t)

	skillPath := filepath.Join(configDir, "my-skill")
	os.WriteFile(skillPath, []byte("skill"), 0644)

	mgr := store.NewManager(configDir)
	cfg, _ := mgr.LoadBaseConfig()
	cfg.Skills = append(cfg.Skills, internal.Skill{ID: "s1", Name: "my-skill", Path: skillPath})
	mgr.SaveBaseConfig(cfg)

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".gemini", "skills"), 0755)

	app.AddSkill(projectDir, "my-skill", "", []string{"gemini"}, false, false)
	err := app.RemoveSkill(projectDir, "my-skill", false)
	assert.NoError(t, err)
}

func TestAddSkillDuplicate(t *testing.T) {
	app, configDir := setupApp(t)

	mgr := store.NewManager(configDir)
	cfg, _ := mgr.LoadBaseConfig()
	cfg.Skills = append(cfg.Skills, internal.Skill{ID: "s1", Name: "dup-skill", RegistryName: "reg1"})
	cfg.Skills = append(cfg.Skills, internal.Skill{ID: "s2", Name: "dup-skill", RegistryName: "reg2"})
	mgr.SaveBaseConfig(cfg)

	projectDir := t.TempDir()

	// Should fail without registry
	err := app.AddSkill(projectDir, "dup-skill", "", []string{"gemini"}, false, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple skills found")
	assert.Contains(t, err.Error(), "reg1, reg2")

	// Should succeed with registry
	os.MkdirAll(filepath.Join(projectDir, ".gemini", "skills"), 0755)
	err = app.AddSkill(projectDir, "dup-skill", "reg1", []string{"gemini"}, false, false)
	assert.NoError(t, err)
}

func TestAddSkillUpdatesSummary(t *testing.T) {
	app, configDir := setupApp(t)

	// Register a skill
	skillPath := filepath.Join(configDir, "my-skill")
	os.WriteFile(skillPath, []byte("skill"), 0644)
	mgr := store.NewManager(configDir)
	cfg, _ := mgr.LoadBaseConfig()
	cfg.Skills = append(cfg.Skills, internal.Skill{ID: "s1", Name: "my-skill", Path: skillPath})
	mgr.SaveBaseConfig(cfg)

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".gemini", "skills"), 0755)

	// Add skill to project
	err := app.AddSkill(projectDir, "my-skill", "", []string{"gemini"}, false, false)
	assert.NoError(t, err)

	// Verify summary was updated
	summary, err := mgr.LoadSummaryConfig()
	assert.NoError(t, err)

	found := false
	for _, ctx := range summary.Contexts {
		if ctx.Dir == projectDir {
			found = true
			break
		}
	}
	assert.True(t, found, "Project directory should be in summary")
}

func TestStopSkill(t *testing.T) {
	app, configDir := setupApp(t)

	skillPath := filepath.Join(configDir, "my-skill")
	os.WriteFile(skillPath, []byte("skill"), 0644)

	mgr := store.NewManager(configDir)
	cfg, _ := mgr.LoadBaseConfig()
	cfg.Skills = append(cfg.Skills, internal.Skill{ID: "s1", Name: "my-skill", Path: skillPath})
	mgr.SaveBaseConfig(cfg)

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".gemini", "skills"), 0755)

	app.AddSkill(projectDir, "my-skill", "", []string{"gemini"}, false, false)
	err := app.StopSkill(projectDir, "my-skill", []string{"gemini"}, false)
	assert.NoError(t, err)
}
