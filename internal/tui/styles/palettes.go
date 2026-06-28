package styles

import "encoding/json"
import "os"
import "path/filepath"

type Config struct {
    ThemeIndex int `json:"theme_index"`
}

func LoadConfig() Config {
    cfg := Config{}
    p := configPath()
    data, err := os.ReadFile(p)
    if err != nil { return cfg }
    json.Unmarshal(data, &cfg)
    if cfg.ThemeIndex < 0 || cfg.ThemeIndex >= len(Themes) { cfg.ThemeIndex = 0 }
    return cfg
}

func SaveConfig(cfg Config) {
    p := configPath()
    os.MkdirAll(filepath.Dir(p), 0700)
    data, _ := json.Marshal(cfg)
    os.WriteFile(p, data, 0600)
}

func configPath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".ans", "tui-config.json")
}
