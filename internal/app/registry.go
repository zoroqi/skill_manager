package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/zoroqi/skill_manager/internal"
)

// RegisterAdd clones a git repository and registers it.
func (a *App) RegisterAdd(url, name, customPath string) error {
	if name == "" {
		name = extractRepoName(url)
	}

	targetPath, err := a.resolveRegisterPath(customPath, name)
	if err != nil {
		return err
	}

	// Clone the repository
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", url, targetPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Load existing registries
	config, err := a.store.LoadRegistriesConfig()
	if err != nil {
		return err
	}

	// Check if already registered
	for _, r := range config.Registries {
		if r.Name == name {
			return fmt.Errorf("registry with name '%s' already exists", name)
		}
	}

	// Add to registries
	config.Registries = append(config.Registries, internal.Registry{
		Name:        name,
		URL:         url,
		LocalPath:   targetPath,
		LastUpdated: time.Now().Format(time.RFC3339),
	})

	if err := a.store.SaveRegistriesConfig(config); err != nil {
		return err
	}

	// Trigger scan
	_, err = a.ScanSkills(targetPath, name)
	return err
}

// RegisterUpdate pulls updates for a registered repository.
func (a *App) RegisterUpdate(name string) error {
	config, err := a.store.LoadRegistriesConfig()
	if err != nil {
		return err
	}

	var registry *internal.Registry
	var idx int
	for i, r := range config.Registries {
		if r.Name == name {
			registry = &config.Registries[i]
			idx = i
			break
		}
	}

	if registry == nil {
		return fmt.Errorf("registry '%s' not found", name)
	}

	// Git pull
	cmd := exec.Command("git", "-C", registry.LocalPath, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	// Update last updated time
	config.Registries[idx].LastUpdated = time.Now().Format(time.RFC3339)
	if err := a.store.SaveRegistriesConfig(config); err != nil {
		return err
	}

	// Trigger re-scan
	_, err = a.ScanSkills(registry.LocalPath, registry.Name)
	return err
}

// RegisterList returns all registered repositories.
func (a *App) RegisterList() ([]internal.Registry, error) {
	config, err := a.store.LoadRegistriesConfig()
	if err != nil {
		return nil, err
	}
	return config.Registries, nil
}

// RegisterRemove unregisters a repository.
func (a *App) RegisterRemove(name string, deleteFiles bool) error {
	config, err := a.store.LoadRegistriesConfig()
	if err != nil {
		return err
	}

	newRegistries := []internal.Registry{}
	var targetPath string
	found := false
	for _, r := range config.Registries {
		if r.Name == name {
			found = true
			targetPath = r.LocalPath
			continue
		}
		newRegistries = append(newRegistries, r)
	}

	if !found {
		return fmt.Errorf("registry '%s' not found", name)
	}

	config.Registries = newRegistries
	if err := a.store.SaveRegistriesConfig(config); err != nil {
		return err
	}

	if deleteFiles && targetPath != "" {
		// Use mv to a backup directory instead of rm, as per global_context
		home, _ := os.UserHomeDir()
		backupDir := filepath.Join(home, ".skill_manager", "backup", time.Now().Format("20060102150405"))
		if err := os.MkdirAll(backupDir, 0755); err == nil {
			os.Rename(targetPath, filepath.Join(backupDir, filepath.Base(targetPath)))
		}
	}

	// Cleanup skills.json (re-scan overall or just remove stale entries)
	// For now, let's just trigger a full scan of all remaining registries and other known paths
	// Actually, the current ScanSkills just adds or updates. It doesn't remove.
	// We might need a "CleanSkills" or a way to remove skills that no longer exist on disk.
	return a.cleanupSkills()
}

func (a *App) cleanupSkills() error {
	cfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return err
	}

	validSkills := []internal.Skill{}
	for _, s := range cfg.Skills {
		if _, err := os.Stat(s.Path); err == nil {
			validSkills = append(validSkills, s)
		}
	}
	cfg.Skills = validSkills
	return a.store.SaveBaseConfig(cfg)
}

func (a *App) resolveRegisterPath(customPath, name string) (string, error) {
	if customPath != "" {
		if !filepath.IsAbs(customPath) {
			abs, err := filepath.Abs(customPath)
			if err != nil {
				return "", err
			}
			customPath = abs
		}
		return customPath, nil
	}

	if envPath := os.Getenv("SKILL_MANAGER_REGISTER_PATH"); envPath != "" {
		return filepath.Join(envPath, name), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, "registries", name), nil
}

func extractRepoName(url string) string {
	url = strings.TrimSuffix(url, ".git")
	// Handle SSH format: git@github.com:org/repo
	url = strings.Replace(url, ":", "/", 1)
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		org := parts[len(parts)-2]
		repo := parts[len(parts)-1]
		// Handle git@github.com part
		if strings.Contains(org, "@") {
			orgParts := strings.Split(org, "@")
			if len(orgParts) > 1 {
				org = orgParts[1]
			}
		}
		return fmt.Sprintf("%s_%s", org, repo)
	}
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}
