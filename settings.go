package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PlayerSetup holds the saved configuration for one player slot.
type PlayerSetup struct {
	Name       string `json:"name"`
	IsComputer bool   `json:"is_computer"`
}

// Settings holds user preferences that are persisted across sessions.
type Settings struct {
	NumPlayers int           `json:"num_players"`
	Players    []PlayerSetup `json:"players"`
	Lang       Lang          `json:"lang"`
	AISpeed    AISpeed       `json:"ai_speed"`
}

func settingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "yatzy_settings.json"
	}
	return filepath.Join(home, ".yatzy_settings.json")
}

func LoadSettings() *Settings {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return nil
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil
	}
	return &s
}

func SaveSettings(s *Settings) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(settingsPath(), data, 0644)
}
