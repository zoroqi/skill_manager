package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	beginMarker = "# BEGIN skill_manager managed"
	endMarker   = "# END skill_manager managed"
)

func updateGitignore(targetDir string, ignorePaths []string) error {
	gitignorePath := filepath.Join(targetDir, ".gitignore")

	var lines []string
	if _, err := os.Stat(gitignorePath); err == nil {
		file, err := os.Open(gitignorePath)
		if err != nil {
			return fmt.Errorf("failed to open .gitignore: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read .gitignore: %w", err)
		}
	}

	// Remove existing managed block
	var newLines []string
	inBlock := false
	for _, line := range lines {
		if strings.TrimSpace(line) == beginMarker {
			inBlock = true
			continue
		}
		if strings.TrimSpace(line) == endMarker {
			inBlock = false
			continue
		}
		if !inBlock {
			newLines = append(newLines, line)
		}
	}

	// Add new managed block if ignorePaths is not empty
	if len(ignorePaths) > 0 {
		// Ensure a blank line before the block if the file is not empty and doesn't end with one
		if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
			newLines = append(newLines, "")
		}
		newLines = append(newLines, beginMarker)
		newLines = append(newLines, ignorePaths...)
		newLines = append(newLines, endMarker)
	}

	// If no lines left (e.g. file was only our block and it's now empty), don't write anything if file didn't exist
	if len(newLines) == 0 {
		if _, err := os.Stat(gitignorePath); err == nil {
			// File exists but is now empty, we can choose to remove it or leave it empty.
			// Usually safer to just write it as empty or remove it.
			return os.WriteFile(gitignorePath, []byte(""), 0644)
		}
		return nil
	}

	content := strings.Join(newLines, "\n")
	// Ensure file ends with a newline
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return os.WriteFile(gitignorePath, []byte(content), 0644)
}
