package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoroqi/skill_manager/internal"
)

func TestCreateAndDeleteGroup(t *testing.T) {
	app, _ := setupApp(t)

	err := app.CreateGroup("g1", "Group 1")
	assert.NoError(t, err)

	groups, err := app.ListGroups()
	assert.NoError(t, err)
	assert.Len(t, groups.Groups, 1)
	assert.Equal(t, "g1", groups.Groups[0].Name)

	err = app.DeleteGroup("g1")
	assert.NoError(t, err)

	groups, err = app.ListGroups()
	assert.NoError(t, err)
	assert.Empty(t, groups.Groups)
}

func TestCreateGroupDuplicate(t *testing.T) {
	app, _ := setupApp(t)
	app.CreateGroup("g1", "")
	err := app.CreateGroup("g1", "")
	assert.Error(t, err)
}

func TestAddRemoveSkillToGroup(t *testing.T) {
	app, _ := setupApp(t)

	app.CreateGroup("g1", "")

	mgr := app.store
	cfg, _ := mgr.LoadBaseConfig()
	cfg.Skills = append(cfg.Skills, internal.Skill{ID: "s1", Name: "my-skill"})
	mgr.SaveBaseConfig(cfg)

	err := app.AddSkillToGroup("g1", []string{"my-skill"}, "")
	assert.NoError(t, err)

	groups, _ := app.ListGroups()
	assert.Contains(t, groups.Groups[0].SkillIDs, "s1")

	err = app.RemoveSkillFromGroup("g1", []string{"my-skill"})
	assert.NoError(t, err)

	groups, _ = app.ListGroups()
	assert.NotContains(t, groups.Groups[0].SkillIDs, "s1")
}
