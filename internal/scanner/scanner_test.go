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

	skills, err := Scan(tempDir)
	assert.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "my-skill", skills[0].Name)
	assert.Equal(t, skillDir, skills[0].Path)
	assert.Equal(t, "universal", skills[0].Metadata["scope"])
	assert.Equal(t, []any{"Read", "Write"}, skills[0].Metadata["allowed-tools"])
	assert.NotEmpty(t, skills[0].ID)
}

func TestGenerateID(t *testing.T) {
	parent := "skills"
	name := "my-skill"
	id := generateID(parent, name)

	// md5("skills/my-skill")
	// "skills/my-skill" -> md5 -> 188a2c229e34c3352abdf057148a070b
	expected := "188a2c229e34c3352abdf057148a070b"
	assert.Equal(t, expected, id)
}
