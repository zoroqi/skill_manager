// internal/scanner/scanner_test.go
package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestScanDirectory(t *testing.T) {
	tempDir := t.TempDir()
	skillDir := filepath.Join(tempDir, "my-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("description"), 0644)

	skills, err := Scan(tempDir)
	assert.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "my-skill", skills[0].Name)
	assert.Equal(t, skillDir, skills[0].Path)
	assert.NotEmpty(t, skills[0].ID) // ID should be hashed
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
