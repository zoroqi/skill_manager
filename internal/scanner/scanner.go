// internal/scanner/scanner.go
package scanner

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"github.com/zoroqi/skill_manager/internal/config"
)

func generateID(parentDir, skillName string) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s/%s", parentDir, skillName)))
	return hex.EncodeToString(hash[:])
}

// Scan traverses subdirectories of rootDir, checks for SKILL.md, and returns discovered skills.
func Scan(rootDir string) ([]config.Skill, error) {
	var skills []config.Skill
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absRoot)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillPath := filepath.Join(absRoot, entry.Name())
			skillMdPath := filepath.Join(skillPath, "SKILL.md")
			if _, err := os.Stat(skillMdPath); err == nil {
				// parentDir is the base name of the directory where the skill directory is located
				parentDir := filepath.Base(filepath.Dir(skillPath))
				id := generateID(parentDir, entry.Name())
				skills = append(skills, config.Skill{
					ID:          id,
					Name:        entry.Name(),
					Path:        skillPath,
					Description: "", // Extracted description can be implemented later
				})
			}
		}
	}
	return skills, nil
}
