// internal/scanner/scanner.go
package scanner

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zoroqi/skill_manager/internal"
	"gopkg.in/yaml.v3"
)

func generateID(registryOrParentDir, skillName string) string {
	hash := md5.Sum(fmt.Appendf(nil, "%s_%s", registryOrParentDir, skillName))
	return hex.EncodeToString(hash[:])
}

func parseSkillMd(path string) (internal.Skill, error) {
	var s internal.Skill
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
func Scan(rootDir string, registryName string) ([]internal.Skill, error) {
	var skills []internal.Skill
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories (starting with .)
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		if !d.IsDir() && d.Name() == "SKILL.md" {
			skillPath := filepath.Dir(path)
			s, err := parseSkillMd(path)
			if err != nil {
				// Log error and skip
				return nil
			}

			// Override or ensure essential fields
			skillDirName := filepath.Base(skillPath)
			if s.Name == "" {
				s.Name = skillDirName
			}
			s.Path = skillPath
			s.RegistryName = registryName

			idKey := registryName
			if idKey == "" {
				idKey = filepath.Base(filepath.Dir(skillPath))
			}
			s.ID = generateID(idKey, s.Name)

			skills = append(skills, s)

			// Once a SKILL.md is found, don't scan deeper into this directory
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return skills, nil
}
