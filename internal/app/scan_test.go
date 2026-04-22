package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
