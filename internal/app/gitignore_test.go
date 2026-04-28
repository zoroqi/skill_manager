package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateGitignore(t *testing.T) {
	t.Run("create new gitignore", func(t *testing.T) {
		tmpDir := t.TempDir()
		ignorePaths := []string{".gemini/skills/", ".claude/skills/"}

		err := updateGitignore(tmpDir, ignorePaths)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
		require.NoError(t, err)

		expected := strings.Join([]string{
			beginMarker,
			".gemini/skills/",
			".claude/skills/",
			endMarker,
			"",
		}, "\n")
		assert.Equal(t, expected, string(content))
	})

	t.Run("update existing gitignore with managed block", func(t *testing.T) {
		tmpDir := t.TempDir()
		initialContent := "node_modules/\n\n" + beginMarker + "\nold/path/\n" + endMarker + "\n"
		err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(initialContent), 0644)
		require.NoError(t, err)

		ignorePaths := []string{".gemini/skills/"}
		err = updateGitignore(tmpDir, ignorePaths)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
		require.NoError(t, err)

		expected := "node_modules/\n\n" + beginMarker + "\n.gemini/skills/\n" + endMarker + "\n"
		assert.Equal(t, expected, string(content))
	})

	t.Run("remove managed block when no paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		initialContent := "node_modules/\n\n" + beginMarker + "\nold/path/\n" + endMarker + "\n"
		err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(initialContent), 0644)
		require.NoError(t, err)

		err = updateGitignore(tmpDir, nil)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
		require.NoError(t, err)

		assert.Equal(t, "node_modules/\n", string(content))
	})

	t.Run("append to gitignore without managed block", func(t *testing.T) {
		tmpDir := t.TempDir()
		initialContent := "node_modules/" // No newline at end
		err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(initialContent), 0644)
		require.NoError(t, err)

		ignorePaths := []string{".gemini/skills/"}
		err = updateGitignore(tmpDir, ignorePaths)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
		require.NoError(t, err)

		expected := "node_modules/\n\n" + beginMarker + "\n.gemini/skills/\n" + endMarker + "\n"
		assert.Equal(t, expected, string(content))
	})
}
