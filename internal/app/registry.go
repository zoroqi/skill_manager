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
	// Load existing registries
	config, err := a.store.LoadRegistriesConfig()
	if err != nil {
		return err
	}

	if name == "" {
		name = extractRepoName(url)
	}

	// Check if already registered
	for _, r := range config.Registries {
		if r.Name == name {
			return fmt.Errorf("registry with name '%s' already exists, please use `register update %s`", name, name)
		}
	}

	logicalPath, physicalPath, err := a.resolveRegisterPath(customPath, name)
	if err != nil {
		return err
	}

	// Clone the repository
	if err := os.MkdirAll(filepath.Dir(physicalPath), 0755); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", url, physicalPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Add to registries
	config.Registries = append(config.Registries, internal.Registry{
		Name:        name,
		URL:         url,
		LocalPath:   logicalPath,
		LastUpdated: time.Now().Format(time.RFC3339),
	})

	if err := a.store.SaveRegistriesConfig(config); err != nil {
		return err
	}

	// Trigger scan
	_, err = a.ScanSkills(physicalPath, name)
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

	physicalPath, err := registry.GetPhysicalPath()
	if err != nil {
		return err
	}

	// Git pull
	cmd := exec.Command("git", "-C", physicalPath, "pull")
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
	_, err = a.ScanSkills(physicalPath, registry.Name)
	return err
}

// RegisterUpdateAll pulls updates for all registered repositories.
func (a *App) RegisterUpdateAll() error {
	config, err := a.store.LoadRegistriesConfig()
	if err != nil {
		return err
	}

	var errs []error
	for _, r := range config.Registries {
		fmt.Printf("Updating %s...\n", r.Name)
		if err := a.RegisterUpdate(r.Name); err != nil {
			errs = append(errs, fmt.Errorf("failed to update %s: %w", r.Name, err))
		}
	}

	if len(errs) > 0 {
		var combined []string
		for _, e := range errs {
			combined = append(combined, e.Error())
		}
		return fmt.Errorf("encountered errors during full update:\n%s", strings.Join(combined, "\n"))
	}

	return nil
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
	var registryToRemove *internal.Registry
	for i, r := range config.Registries {
		if r.Name == name {
			registryToRemove = &config.Registries[i]
			continue
		}
		newRegistries = append(newRegistries, r)
	}

	if registryToRemove == nil {
		return fmt.Errorf("registry '%s' not found", name)
	}

	physicalPath, err := registryToRemove.GetPhysicalPath()
	if err != nil {
		return err
	}

	config.Registries = newRegistries
	if err := a.store.SaveRegistriesConfig(config); err != nil {
		return err
	}

	if deleteFiles && physicalPath != "" {
		// Use mv to a backup directory instead of rm, as per global_context
		home, _ := os.UserHomeDir()
		backupDir := filepath.Join(home, ".skill_manager", "backup", time.Now().Format("20060102150405"))
		if err := os.MkdirAll(backupDir, 0755); err == nil {
			os.Rename(physicalPath, filepath.Join(backupDir, filepath.Base(physicalPath)))
		}
	}

	// Cleanup skills.json and groups.json
	return a.cleanupSkills()
}

func (a *App) cleanupSkills() error {
	registriesCfg, err := a.store.LoadRegistriesConfig()
	if err != nil {
		return err
	}

	regMap := make(map[string]string)
	for _, r := range registriesCfg.Registries {
		if p, err := r.GetPhysicalPath(); err == nil {
			regMap[r.Name] = p
		}
	}

	cfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return err
	}

	validSkills := []internal.Skill{}
	validSkillIDs := make(map[string]bool)
	var newlyDeleted []internal.Skill

	for _, s := range cfg.Skills {
		var skillMdPath string
		isValid := false
		if s.RegistryName != "" {
			physicalBase, ok := regMap[s.RegistryName]
			if ok {
				if filepath.IsAbs(s.Path) {
					skillMdPath = filepath.Join(s.Path, "SKILL.md")
				} else {
					skillMdPath = filepath.Join(physicalBase, s.Path, "SKILL.md")
				}
				if _, err := os.Stat(skillMdPath); err == nil {
					isValid = true
				}
			}
		} else {
			skillMdPath = filepath.Join(s.Path, "SKILL.md")
			if _, err := os.Stat(skillMdPath); err == nil {
				isValid = true
			}
		}

		if isValid {
			validSkills = append(validSkills, s)
			validSkillIDs[s.ID] = true
		} else {
			newlyDeleted = append(newlyDeleted, s)
		}
	}

	deletedMap := make(map[string]internal.Skill)
	for _, d := range cfg.DeletedSkills {
		deletedMap[d.ID] = d
	}
	for _, r := range newlyDeleted {
		deletedMap[r.ID] = r
	}
	for _, v := range validSkills {
		delete(deletedMap, v.ID)
	}

	newDeleted := make([]internal.Skill, 0, len(deletedMap))
	for _, d := range deletedMap {
		newDeleted = append(newDeleted, d)
	}
	cfg.DeletedSkills = newDeleted
	cfg.Skills = validSkills
	if err := a.store.SaveBaseConfig(cfg); err != nil {
		return err
	}

	// Clean up invalid skill IDs in groups.json
	groupsCfg, err := a.store.LoadGroupsConfig()
	if err == nil {
		changed := false
		for i, g := range groupsCfg.Groups {
			var newIDs []string
			for _, id := range g.SkillIDs {
				if validSkillIDs[id] {
					newIDs = append(newIDs, id)
				} else {
					changed = true
				}
			}
			groupsCfg.Groups[i].SkillIDs = newIDs
		}
		if changed {
			_ = a.store.SaveGroupsConfig(groupsCfg)
		}
	}

	return nil
}

func (a *App) resolveRegisterPath(customPath, name string) (string, string, error) {
	if customPath != "" {
		if !filepath.IsAbs(customPath) {
			abs, err := filepath.Abs(customPath)
			if err != nil {
				return "", "", err
			}
			customPath = abs
		}
		return customPath, customPath, nil
	}

	logicalPath := filepath.Join("$"+internal.ENV_REGISTER_PATH, name)

	if envPath := os.Getenv(internal.ENV_REGISTER_PATH); envPath != "" {
		return logicalPath, filepath.Join(envPath, name), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	return logicalPath, filepath.Join(cwd, "registries", name), nil
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
