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
