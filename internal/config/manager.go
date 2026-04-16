package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Manager struct {
	configDir string
}

func NewManager(configDir string) *Manager {
	return &Manager{configDir: configDir}
}

func (m *Manager) Init() error {
	if err := os.MkdirAll(m.configDir, 0755); err != nil {
		return err
	}
	base := BaseConfig{Version: "1.0", Skills: []Skill{}}
	return m.SaveBaseConfig(base)
}

func (m *Manager) SaveBaseConfig(cfg BaseConfig) error {
	path := filepath.Join(m.configDir, "skills.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *Manager) LoadBaseConfig() (BaseConfig, error) {
	path := filepath.Join(m.configDir, "skills.json")
	var cfg BaseConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return BaseConfig{Version: "1.0", Skills: []Skill{}}, nil
		}
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}
