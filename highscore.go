package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// HighScoreEntry is one record in the top-scores table.
type HighScoreEntry struct {
	Name       string    `json:"name"`
	Score      int       `json:"score"`
	NumPlayers int       `json:"num_players"`
	Date       time.Time `json:"date"`
}

// PlayerStats holds per-name lifetime statistics.
type PlayerStats struct {
	Wins      int `json:"wins"`
	Losses    int `json:"losses"`
	BestScore int `json:"best_score"`
}

func (s PlayerStats) Games() int { return s.Wins + s.Losses }

func (s PlayerStats) WinRate() float64 {
	if s.Games() == 0 {
		return 0
	}
	return float64(s.Wins) / float64(s.Games()) * 100
}

// HighScores is the persistent collection of entries and per-player stats.
type HighScores struct {
	Entries []HighScoreEntry       `json:"entries"`
	Stats   map[string]PlayerStats `json:"stats"`
}

func highScorePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "kniffel_highscores.json"
	}
	return filepath.Join(home, ".kniffel_highscores.json")
}

func LoadHighScores() *HighScores {
	data, err := os.ReadFile(highScorePath())
	if err != nil {
		return &HighScores{Stats: make(map[string]PlayerStats)}
	}
	var hs HighScores
	if err := json.Unmarshal(data, &hs); err != nil {
		return &HighScores{Stats: make(map[string]PlayerStats)}
	}
	if hs.Stats == nil {
		hs.Stats = make(map[string]PlayerStats)
	}
	return &hs
}

func (hs *HighScores) Save() {
	data, err := json.MarshalIndent(hs, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(highScorePath(), data, 0644)
}

func (hs *HighScores) AddEntry(entry HighScoreEntry) {
	hs.Entries = append(hs.Entries, entry)
	sort.Slice(hs.Entries, func(i, j int) bool {
		return hs.Entries[i].Score > hs.Entries[j].Score
	})
	if len(hs.Entries) > 20 {
		hs.Entries = hs.Entries[:20]
	}
}

func (hs *HighScores) UpdateStats(name string, score int, won bool) {
	if hs.Stats == nil {
		hs.Stats = make(map[string]PlayerStats)
	}
	s := hs.Stats[name]
	if won {
		s.Wins++
	} else {
		s.Losses++
	}
	if score > s.BestScore {
		s.BestScore = score
	}
	hs.Stats[name] = s
}

// SortedStats returns player stats sorted by wins descending.
func (hs *HighScores) SortedStats() []struct {
	Name  string
	Stats PlayerStats
} {
	result := make([]struct {
		Name  string
		Stats PlayerStats
	}, 0, len(hs.Stats))
	for name, s := range hs.Stats {
		result = append(result, struct {
			Name  string
			Stats PlayerStats
		}{name, s})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Stats.Wins != result[j].Stats.Wins {
			return result[i].Stats.Wins > result[j].Stats.Wins
		}
		return result[i].Stats.BestScore > result[j].Stats.BestScore
	})
	return result
}
