// internal/store/agent.go
package store

import (
	"fmt"
	"os"
	"path/filepath"
)

var SupportedAgents = []string{"gemini", "claude", "codex", "opencode", "agents"}

func GetAgentDir(agent string) string {
	if agent == "agents" {
		return ".agents"
	}
	return "." + agent
}

func EnsureAgentSkillsDir(targetDir, agent string) (string, error) {
	agentDir := GetAgentDir(agent)
	skillsDir := filepath.Join(targetDir, agentDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create agent skills directory %s: %w", skillsDir, err)
	}
	return skillsDir, nil
}

func FindExistingAgents(targetDir string) []string {
	var existing []string
	for _, a := range SupportedAgents {
		if info, err := os.Stat(filepath.Join(targetDir, GetAgentDir(a))); err == nil && info.IsDir() {
			existing = append(existing, a)
		}
	}
	return existing
}

func RemoveAgentSymlinks(targetDir, skillName string, agents []string) error {
	if len(agents) == 0 {
		agents = FindExistingAgents(targetDir)
	}
	for _, a := range agents {
		skillsDir := filepath.Join(targetDir, GetAgentDir(a), "skills")
		linkPath := filepath.Join(skillsDir, skillName)

		if _, err := os.Lstat(linkPath); err == nil {
			if err := os.Remove(linkPath); err != nil {
				return fmt.Errorf("failed to remove symlink %s: %w", linkPath, err)
			}
		}
	}
	return nil
}
