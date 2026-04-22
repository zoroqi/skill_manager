// internal/scanner/scanner_test.go
package scanner

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestScanDirectory(t *testing.T) {
	tempDir := t.TempDir()
	skillDir := filepath.Join(tempDir, "my-skill")
	os.MkdirAll(skillDir, 0755)

	yamlContent := `---
name: my-skill
scope: universal
description: A powerful skill
allowed-tools: [Read, Write]
---
# Instructions`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(yamlContent), 0644)

	skills, err := Scan(tempDir, "")
	assert.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "my-skill", skills[0].Name)
	assert.Equal(t, skillDir, skills[0].Path)
	assert.Equal(t, "universal", skills[0].Metadata["scope"])
	assert.Equal(t, []any{"Read", "Write"}, skills[0].Metadata["allowed-tools"])
	assert.NotEmpty(t, skills[0].ID)
}

func TestScanDirectoryWithRegistry(t *testing.T) {
	tempDir := t.TempDir()
	skillDir := filepath.Join(tempDir, "my-skill")
	os.MkdirAll(skillDir, 0755)

	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# my-skill"), 0644)

	registryName := "org_repo"
	skills, err := Scan(tempDir, registryName)
	assert.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, registryName, skills[0].RegistryName)

	// ID should be md5("org_repo_my-skill")
	// echo -n "org_repo_my-skill" | md5
	expectedID := "4118c65208ab5123e7be199e209dc454"
	assert.Equal(t, expectedID, skills[0].ID)
}

func TestScanDirectoryRecursive(t *testing.T) {
	tempDir := t.TempDir()

	// Skill 1 in root
	skill1Dir := filepath.Join(tempDir, "skill1")
	os.MkdirAll(skill1Dir, 0755)
	os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte("# Skill 1"), 0644)

	// Nested SKILL.md inside skill1 (should be skipped)
	nestedDir := filepath.Join(skill1Dir, "nested")
	os.MkdirAll(nestedDir, 0755)
	os.WriteFile(filepath.Join(nestedDir, "SKILL.md"), []byte("# Nested Skill"), 0644)

	// Skill 2 in skills/ subdirectory
	skill2Dir := filepath.Join(tempDir, "skills", "skill2")
	os.MkdirAll(skill2Dir, 0755)
	os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte("# Skill 2"), 0644)

	// Hidden directory (should be skipped)
	hiddenDir := filepath.Join(tempDir, ".git", "hooks")
	os.MkdirAll(hiddenDir, 0755)
	os.WriteFile(filepath.Join(hiddenDir, "SKILL.md"), []byte("# Hidden Skill"), 0644)

	skills, err := Scan(tempDir, "test_repo")
	assert.NoError(t, err)
	assert.Len(t, skills, 2)

	skillNames := []string{skills[0].Name, skills[1].Name}
	assert.Contains(t, skillNames, "skill1")
	assert.Contains(t, skillNames, "skill2")
	assert.NotContains(t, skillNames, "Nested Skill")
	assert.NotContains(t, skillNames, "Hidden Skill")
}

func TestGenerateID(t *testing.T) {
	parent := "skills"
	name := "my-skill"
	id := generateID(parent, name)

	// md5("skills_my-skill")
	// echo -n "skills_my-skill" | md5
	expected := "4618487b71c39a2e270de658a7381c0f"
	assert.Equal(t, expected, id)
}
