package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

func TestRefactoredLogic(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Setup Config
	configDir := filepath.Join(tempDir, ".skill_manager")
	os.Setenv("SKILL_MANAGER_CONFIG", configDir)
	defer os.Unsetenv("SKILL_MANAGER_CONFIG")

	manager := store.NewManager(configDir)
	err := manager.Init()
	assert.NoError(t, err)

	// Create dummy skills
	s1Path := filepath.Join(tempDir, "s1")
	os.MkdirAll(s1Path, 0755)
	s2Path := filepath.Join(tempDir, "s2")
	os.MkdirAll(s2Path, 0755)

	err = manager.SaveBaseConfig(internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{ID: "id1", Name: "skill1", Path: s1Path},
			{ID: "id2", Name: "skill1", Path: s2Path, RegistryName: "reg2"}, // Duplicate name
			{ID: "id3", Name: "skill2", Path: s2Path},
		},
	})
	assert.NoError(t, err)

	// Create a group
	err = manager.SaveGroupsConfig(internal.GroupsConfig{
		Version: internal.CONFIG_VERSION,
		Groups: []internal.Group{
			{Name: "groupB", SkillIDs: []string{"id1"}},
			{Name: "groupA", SkillIDs: []string{"id3"}},
		},
	})
	assert.NoError(t, err)

	// 2. Project Setup
	projectDir := filepath.Join(tempDir, "project")
	os.MkdirAll(projectDir, 0755)
	oldWd, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(oldWd)

	// Reset globals
	addAgents = ""
	addGlobal = false
	addGroup = ""
	removeAgents = ""
	removeGlobal = false
	removeGroup = ""

	// 3. Test Add + Group Priority
	// Add groupB (contains id1 "skill1")
	addAgents = "gemini"
	addGroup = "groupB"
	rootCmd.SetArgs([]string{"add", "--group", "groupB", "--agent", "gemini"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify linked
	link1 := filepath.Join(projectDir, ".gemini", "skills", "skill1")
	_, err = os.Lstat(link1)
	assert.NoError(t, err)

	// Add groupA (contains id3 "skill2")
	addGroup = "groupA"
	rootCmd.SetArgs([]string{"add", "--group", "groupA", "--agent", "gemini"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Now add a user skill that conflicts with groupB's skill1
	// reg2 has skill1 too.
	addAgents = "gemini"
	addGroup = ""
	addRegister = "reg2"
	addForce = true
	rootCmd.SetArgs([]string{"add", "skill1", "--agent", "gemini", "--register", "reg2", "--force"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify symlink points to s2Path (reg2) now, because User > Group
	target, _ := os.Readlink(link1)
	assert.Contains(t, target, "s2")

	// 4. Test Remove with Group Masking
	rootCmd.SetArgs([]string{"remove", "skill1"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify link removed
	_, err = os.Lstat(link1)
	assert.True(t, os.IsNotExist(err))

	// Verify InactiveSkills masking
	ctxCfg, _ := store.EnsureLocalContext(projectDir)
	found := false
	for _, in := range ctxCfg.InactiveSkills {
		if in.Name == "skill1" {
			found = true
			break
		}
	}
	// skill1 was provided by groupB (id1) and user (id2).
	// Both matching IDs should be removed from ActiveSkills.
	// id1 was in groupB, so it should be in InactiveSkills.
	assert.True(t, found, "skill1 should be masked in InactiveSkills because it's in groupB")

	// 5. Test Group-vs-Group Priority
	// Create another group that also has a skill named "skill2"
	s3Path := filepath.Join(tempDir, "s3")
	os.MkdirAll(s3Path, 0755)
	err = manager.SaveBaseConfig(internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{ID: "id1", Name: "skill1", Path: s1Path},
			{ID: "id2", Name: "skill1", Path: s2Path, RegistryName: "reg2"},
			{ID: "id3", Name: "skill2", Path: s2Path},
			{ID: "id4", Name: "skill2", Path: s3Path, RegistryName: "reg3"},
		},
	})
	assert.NoError(t, err)

	err = manager.SaveGroupsConfig(internal.GroupsConfig{
		Version: internal.CONFIG_VERSION,
		Groups: []internal.Group{
			{Name: "groupB", SkillIDs: []string{"id1", "id4"}}, // groupB has id4 (skill2)
			{Name: "groupA", SkillIDs: []string{"id3"}},        // groupA has id3 (skill2)
		},
	})
	assert.NoError(t, err)

	// Currently groupA and groupB are active.
	// Since groupA < groupB alphabetically, groupA's skill2 (id3/s2Path) should win.
	syncGlobal = false
	rootCmd.SetArgs([]string{"sync"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	link2 := filepath.Join(projectDir, ".gemini", "skills", "skill2")
	target, _ = os.Readlink(link2)
	assert.Contains(t, target, "s2", "groupA should win over groupB for skill2")

	// 6. Test Status Output
	rootCmd.SetArgs([]string{"status"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// 7. Test partial removal with --agent
	// Currently skill2 is active for gemini via groupA.
	// Let's add it for claude too via user.
	addAgents = "claude"
	addRegister = "reg3"
	addForce = false
	rootCmd.SetArgs([]string{"add", "skill2", "--agent", "claude", "--register", "reg3"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify both exist
	_, err = os.Lstat(filepath.Join(projectDir, ".claude", "skills", "skill2"))
	assert.NoError(t, err)

	// Remove only gemini agent for skill2
	removeAgents = "gemini"
	rootCmd.SetArgs([]string{"remove", "skill2", "--agent", "gemini"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Verify gemini link removed, claude link remains
	_, err = os.Lstat(filepath.Join(projectDir, ".gemini", "skills", "skill2"))
	assert.True(t, os.IsNotExist(err), "gemini link should be removed")
	_, err = os.Lstat(filepath.Join(projectDir, ".claude", "skills", "skill2"))
	assert.NoError(t, err, "claude link should remain")

	// Verify InactiveSkills masking for skill2/gemini
	ctxCfg, _ = store.EnsureLocalContext(projectDir)
	foundMask := false
	for _, in := range ctxCfg.InactiveSkills {
		if in.Name == "skill2" {
			assert.Contains(t, in.Agents, "gemini")
			foundMask = true
			break
		}
	}
	assert.True(t, foundMask)
}

func TestStatusCommand_WarningForMissingSkills(t *testing.T) {
	tempDir := t.TempDir()

	configDir := filepath.Join(tempDir, ".skill_manager")
	os.Setenv("SKILL_MANAGER_CONFIG", configDir)
	defer os.Unsetenv("SKILL_MANAGER_CONFIG")

	manager := store.NewManager(configDir)
	err := manager.Init()
	assert.NoError(t, err)

	s1Path := filepath.Join(tempDir, "s1")
	os.MkdirAll(s1Path, 0755)
	os.WriteFile(filepath.Join(s1Path, "SKILL.md"), []byte("# valid-skill"), 0644)

	err = manager.SaveBaseConfig(internal.BaseConfig{
		Version: internal.CONFIG_VERSION,
		Skills: []internal.Skill{
			{ID: "valid_id", Name: "valid_skill", Path: s1Path},
		},
		DeletedSkills: []internal.Skill{
			{ID: "missing_id_123", Name: "my_removed_skill", RegistryName: "org_repo"},
		},
	})
	assert.NoError(t, err)

	projectDir := filepath.Join(tempDir, "project")
	os.MkdirAll(projectDir, 0755)
	oldWd, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(oldWd)

	// Set context with missing skill and missing group
	ctxCfg := internal.ContextConfig{
		Version: internal.CONFIG_VERSION,
		Scope:   "test-proj",
		ActiveSkills: []internal.SkillUsage{
			{ID: "valid_id", Agents: []string{"gemini"}},
			{ID: "missing_id_123", Agents: []string{"gemini", "claude"}},
		},
		ActiveGroups: []internal.GroupUsage{
			{Name: "missing_group_xyz", Agents: []string{"gemini"}},
		},
	}
	assert.NoError(t, store.SaveLocalContext(projectDir, ctxCfg))

	// Run status
	statusGlobal = false
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"status"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "valid_skill")
	assert.Contains(t, out, "Warning: 1 active skill(s) missing or deleted from skills library:")
	assert.Contains(t, out, "my_removed_skill (registry: org_repo, id: missing_id_123) (agents: gemini, claude)")
	assert.Contains(t, out, "Warning: 1 active group(s) missing or deleted from groups library:")
	assert.Contains(t, out, "missing_group_xyz (agents: gemini)")
	assert.Contains(t, out, "Please run `skill_manager sync` to clean up invalid skills.")

	// Test Global status warning
	globalCtx := internal.ContextConfig{
		Version: internal.CONFIG_VERSION,
		Scope:   "global",
		ActiveSkills: []internal.SkillUsage{
			{ID: "global_missing_id", Agents: []string{}},
		},
	}
	assert.NoError(t, manager.SaveContextConfig(globalCtx, internal.FILENAME_GLOBAL_CONTEXT))

	statusGlobal = true
	bufGlobal := new(bytes.Buffer)
	rootCmd.SetOut(bufGlobal)
	rootCmd.SetArgs([]string{"status", "-g"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	outGlobal := bufGlobal.String()
	assert.Contains(t, outGlobal, "Warning: 1 active skill(s) missing or deleted from skills library:")
	assert.Contains(t, outGlobal, "global_missing_id (agents: all agents)")
	assert.Contains(t, outGlobal, "Please run `skill_manager sync -g` to clean up invalid skills.")
}
