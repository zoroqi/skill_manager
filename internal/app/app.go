package app

import "github.com/zoroqi/skill_manager/internal/store"

type App struct {
	store *store.Manager
}

func New(configDir string) *App {
	return &App{store: store.NewManager(configDir)}
}

// DefaultConfigDir returns the default configuration directory.
func DefaultConfigDir() (string, error) {
	return store.GetDefaultConfigDir()
}
