package internal

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type ThemeConfig struct {
	Background string `json:"background"`
	Text       string `json:"text"`
	Chrome     string `json:"chrome"`
	Accent     string `json:"accent"`
	Muted      string `json:"muted"`
}

type TypographyConfig struct {
	FontPath string  `json:"font_path"`
	FontSize float32 `json:"font_size"`
}

type LayoutConfig struct {
	PagePadding  float32 `json:"page_padding"`
	ContentWidth float32 `json:"content_width"`
}

type UIConfig struct {
	TopBarVisible  bool    `json:"top_bar_visible"`
	TopBarHeight   float32 `json:"top_bar_height"`
	IconSize       float32 `json:"icon_size"`
	ShowNoteName   bool    `json:"show_note_name"`
	ToolbarVisible bool    `json:"toolbar_visible"`
	HomeOnStart    bool    `json:"home_on_start"`
}

type MotionConfig struct {
	LerpRate float32 `json:"lerp_rate"`
	Enabled  bool    `json:"enabled"`
}

type SessionConfig struct {
	VaultPath string `json:"vault_path"`
	LastNote  string `json:"last_note"`
}

type Config struct {
	Theme      ThemeConfig      `json:"theme"`
	Typography TypographyConfig `json:"typography"`
	Layout     LayoutConfig     `json:"layout"`
	UI         UIConfig         `json:"ui"`
	Motion     MotionConfig     `json:"motion"`
	Session    SessionConfig    `json:"session"`
}

func defaultConfig() Config {
	home, _ := os.UserHomeDir()
	vault := filepath.Join(home, "JustJournal")
	return Config{
		Theme: ThemeConfig{
			Background: "#FFFFFF",
			Text:       "#202124",
			Chrome:     "#F7F7F5",
			Accent:     "#DCE6FF",
			Muted:      "#7B8188",
		},
		Typography: TypographyConfig{
			FontSize: 18,
		},
		Layout: LayoutConfig{
			PagePadding:  28,
			ContentWidth: 980,
		},
		UI: UIConfig{
			TopBarVisible:  true,
			TopBarHeight:   34,
			IconSize:       16,
			ShowNoteName:   true,
			ToolbarVisible: true,
		},
		Motion: MotionConfig{
			LerpRate: 10,
			Enabled:  true,
		},
		Session: SessionConfig{
			VaultPath: vault,
			LastNote:  "untitled.md",
		},
	}
}

func configDir() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "justjournal"), nil
}

func configPath() (string, error) {
	root, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "config.json"), nil
}

func loadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg := defaultConfig()
		return cfg, saveConfig(cfg)
	}
	if err != nil {
		return Config{}, err
	}
	cfg := defaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func saveConfig(cfg Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
