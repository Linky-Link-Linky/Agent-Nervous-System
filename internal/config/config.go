package config

import (
	"encoding/json"
	"os"
	path "path/filepath"
)

type TUIConfig struct {
	RefreshIntervalMS int `json:"refresh_interval_ms"`
}

func Default() *TUIConfig {
	return &TUIConfig{RefreshIntervalMS: 2000}
}

func (c *TUIConfig) Path() string {
	home, _ := os.UserHomeDir()
	return path.Join(home, ".ans", "tui-config.json")
}

func Load() *TUIConfig {
	cfg := Default()
	data, err := os.ReadFile(cfg.Path())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	return cfg
}

func (c *TUIConfig) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	dir := path.Dir(c.Path())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(c.Path(), data, 0644)
}

