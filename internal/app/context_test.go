package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStatus(t *testing.T) {
	app, _ := setupApp(t)
	workDir := t.TempDir()

	ctxCfg, skills, groups, err := app.GetStatus(workDir, false)
	assert.NoError(t, err)
	assert.Empty(t, skills)
	assert.Empty(t, groups)
	assert.Equal(t, "project", ctxCfg.Scope)
}

func TestSyncContext(t *testing.T) {
	app, _ := setupApp(t)
	workDir := t.TempDir()

	err := app.SyncContext(workDir, false, false)
	assert.NoError(t, err)
}
