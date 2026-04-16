// internal/scanner/scanner.go
package scanner

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zoroqi/skill_manager/internal/config"
	"gopkg.in/yaml.v3"
)

func generateID(parentDir, skillName string) string {
	hash := md5.Sum(fmt.Appendf(nil, "%s/%s", parentDir, skillName))
	return hex.EncodeToString(hash[:])
}

func parseSkillMd(path string) (config.Skill, error) {
	var s config.Skill
	content, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}

	parts := strings.Split(string(content), "---")
	if len(parts) < 3 {
		// Fallback: Use existing logic if no YAML frontmatter
		lines := strings.SplitSeq(string(content), "\n")
		for line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				s.Description = trimmed
				break
			}
		}
		return s, nil
	}

	err = yaml.Unmarshal([]byte(parts[1]), &s)
	return s, err
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
				s, err := parseSkillMd(skillMdPath)
				if err != nil {
					// Log error and skip or continue
					continue
				}

				// Override or ensure essential fields
				if s.Name == "" {
					s.Name = entry.Name()
				}
				s.Path = skillPath
				parentDir := filepath.Base(filepath.Dir(skillPath))
				s.ID = generateID(parentDir, entry.Name())

				skills = append(skills, s)
			}
		}
	}
	return skills, nil
}
