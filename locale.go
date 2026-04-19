package main

// Lang selects the active UI language.
type Lang int

const (
	LangDE Lang = iota
	LangEN
)

// Locale holds every user-visible string.
type Locale struct {
	// ---- Setup ----
	AppTitle       string
	LabelNumPlayer string
	LabelAISpeed   string
	SpeedSlow      string
	SpeedNormal    string
	SpeedFast      string
	PlayerLabel    string // "Spieler %d:" / "Player %d:"
	DefaultName    string // "Spieler %d"  / "Player %d"
	NameHint       string
	Human          string
	Computer       string
	StartGame      string
	Highscores     string
	LabelLanguage  string

	// ---- Game ----
	ComputerSuffix  string // appended to name in top bar
	CurrentPlayer   string // "Am Zug: %s"
	RollsLeft       string // "Würfe übrig: %d"
	ComputerPlaying string
	RollBtn         string // "Würfeln (%d übrig)"

	// ---- Score categories (13) ----
	Categories [13]string

	// ---- Score sheet ----
	ColCategory  string
	UpperSection string
	Subtotal     string
	BonusRow     string // full label with condition
	LowerSection string
	GrandTotal   string

	// ---- Game over ----
	GameOverTitle string
	ResultSingle  string // single player: "Ergebnis: %d Punkte"
	ResultWinner  string // "Gewinner: %s mit %d Punkten!"
	FinalUpper    string
	FinalBonus    string
	FinalLower    string
	FinalTotal    string
	NewGame       string

	// ---- About ----
	About           string
	AboutTitle      string
	AboutDesc       string
	AboutSourceCode string
	AboutGitHub     string // the URL label
	AboutLicense    string

	// ---- High scores ----
	TabTopScores string
	TabStats     string
	ColRank      string
	ColName      string
	ColScore     string
	ColPlayers   string
	ColDate      string
	NoEntries    string
	Back         string
	ColGames     string
	ColWins      string
	ColLosses    string
	ColWinRate   string
	ColBestScore string
	NoStats      string
}

var locales = [2]Locale{
	// ---- German ----
	{
		AppTitle:       "Yatzy",
		LabelNumPlayer: "Anzahl Spieler:",
		LabelAISpeed:   "Computer-Geschwindigkeit:",
		SpeedSlow:      "Langsam",
		SpeedNormal:    "Normal",
		SpeedFast:      "Schnell",
		PlayerLabel:    "Spieler %d:",
		DefaultName:    "Spieler %d",
		NameHint:       "Name eingeben…",
		Human:          "Mensch",
		Computer:       "Computer",
		StartGame:      "Spiel starten",
		Highscores:     "Highscores",
		LabelLanguage:  "Sprache:",

		ComputerSuffix:  " (Computer)",
		CurrentPlayer:   "Am Zug: %s",
		RollsLeft:       "Würfe übrig: %d",
		ComputerPlaying: "Computer spielt…",
		RollBtn:         "Würfeln (%d übrig)",

		Categories: [13]string{
			"Einser (1er)", "Zweier (2er)", "Dreier (3er)",
			"Vierer (4er)", "Fünfer (5er)", "Sechser (6er)",
			"Dreierpasch", "Viererpasch", "Full House",
			"Kleine Straße", "Große Straße", "Yatzy", "Chance",
		},

		ColCategory:  "Kategorie",
		UpperSection: "─── OBERER TEIL ───",
		Subtotal:     "Zwischensumme",
		BonusRow:     "Bonus (+35 wenn ≥63)",
		LowerSection: "─── UNTERER TEIL ───",
		GrandTotal:   "GESAMTERGEBNIS",

		GameOverTitle: "Spiel beendet!",
		ResultSingle:  "Ergebnis: %d Punkte",
		ResultWinner:  "Gewinner: %s mit %d Punkten!",
		FinalUpper:    "Oberer Teil",
		FinalBonus:    "Bonus",
		FinalLower:    "Unterer Teil",
		FinalTotal:    "GESAMT",
		NewGame:       "Neues Spiel",

		About:           "Über",
		AboutTitle:      "Über Yatzy",
		AboutDesc:       "Eine freie Umsetzung des Würfelspiels Yatzy\nfür bis zu 4 Spieler mit Computer-Gegnern,\nHighscore-Tabelle und Statistiken.",
		AboutSourceCode: "Quellcode:",
		AboutGitHub:     "github.com/Kellerman81/go_yatzy",
		AboutLicense:    "Veröffentlicht unter der MIT-Lizenz.",

		TabTopScores: "Top-Ergebnisse",
		TabStats:     "Statistiken",
		ColRank:      "#",
		ColName:      "Name",
		ColScore:     "Punkte",
		ColPlayers:   "Spieler",
		ColDate:      "Datum",
		NoEntries:    "Noch keine Einträge.",
		Back:         "Zurück",
		ColGames:     "Spiele",
		ColWins:      "Siege",
		ColLosses:    "Niederlagen",
		ColWinRate:   "Siegrate",
		ColBestScore: "Bestpunktzahl",
		NoStats:      "Noch keine Statistiken.",
	},

	// ---- English ----
	{
		AppTitle:       "Yatzy",
		LabelNumPlayer: "Number of players:",
		LabelAISpeed:   "Computer speed:",
		SpeedSlow:      "Slow",
		SpeedNormal:    "Normal",
		SpeedFast:      "Fast",
		PlayerLabel:    "Player %d:",
		DefaultName:    "Player %d",
		NameHint:       "Enter name…",
		Human:          "Human",
		Computer:       "Computer",
		StartGame:      "Start game",
		Highscores:     "High scores",
		LabelLanguage:  "Language:",

		ComputerSuffix:  " (Computer)",
		CurrentPlayer:   "Turn: %s",
		RollsLeft:       "Rolls left: %d",
		ComputerPlaying: "Computer is playing…",
		RollBtn:         "Roll (%d left)",

		Categories: [13]string{
			"Ones", "Twos", "Threes",
			"Fours", "Fives", "Sixes",
			"Three of a Kind", "Four of a Kind", "Full House",
			"Small Straight", "Large Straight", "Yatzy", "Chance",
		},

		ColCategory:  "Category",
		UpperSection: "─── UPPER SECTION ───",
		Subtotal:     "Subtotal",
		BonusRow:     "Bonus (+35 if ≥63)",
		LowerSection: "─── LOWER SECTION ───",
		GrandTotal:   "GRAND TOTAL",

		GameOverTitle: "Game over!",
		ResultSingle:  "Score: %d points",
		ResultWinner:  "Winner: %s with %d points!",
		FinalUpper:    "Upper section",
		FinalBonus:    "Bonus",
		FinalLower:    "Lower section",
		FinalTotal:    "TOTAL",
		NewGame:       "New game",

		About:           "About",
		AboutTitle:      "About Yatzy",
		AboutDesc:       "A free implementation of the dice game Yatzy\nfor up to 4 players with computer opponents,\nhigh scores, and statistics.",
		AboutSourceCode: "Source code:",
		AboutGitHub:     "github.com/Kellerman81/go_yatzy",
		AboutLicense:    "Released under the MIT License.",

		TabTopScores: "Top scores",
		TabStats:     "Statistics",
		ColRank:      "#",
		ColName:      "Name",
		ColScore:     "Score",
		ColPlayers:   "Players",
		ColDate:      "Date",
		NoEntries:    "No entries yet.",
		Back:         "Back",
		ColGames:     "Games",
		ColWins:      "Wins",
		ColLosses:    "Losses",
		ColWinRate:   "Win rate",
		ColBestScore: "Best score",
		NoStats:      "No statistics yet.",
	},
}
