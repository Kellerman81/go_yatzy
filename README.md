# go_yatzy

A free implementation of the classic dice game Yatzy (also known as Kniffel / Yahtzee), written in Go using the [Gio](https://gioui.org) UI toolkit.

## Features

### Gameplay
- Classic Yatzy rules with all 13 scoring categories: Ones through Sixes, Three of a Kind, Four of a Kind, Full House, Small Straight, Large Straight, Yatzy, and Chance
- Upper section bonus: +35 points when the upper section total reaches 63 or more
- Up to 4 players per game — any mix of human and computer players
- 3 rolls per turn; keep individual dice between rolls by tapping/clicking them

### Computer opponents
- AI evaluates potential across all available categories and selects the best dice to keep
- Three configurable AI speeds: Slow, Normal, Fast

### High scores & statistics
- Top 20 all-time scores stored locally, showing player name, score, number of players, and date
- Per-player lifetime statistics: games played, wins, losses, win rate, and best score

### Localization
- Full UI available in **English** and **German**, switchable in the setup screen

### Settings persistence
- Player names, human/computer assignments, number of players, language, and AI speed are saved and restored across sessions

## Building

### Desktop (Linux / Windows)

```bash
go build -o go_yatzy .
```

For a Windows GUI binary (no console window):

```bash
go build -ldflags="-H windowsgui" -o go_yatzy.exe .
```

### Android & iOS

Gio supports mobile targets. Follow the platform-specific setup instructions at:

**https://gioui.org/doc/install**

Then build with the `gogio` tool:

```bash
# Android
gogio -target android -o go_yatzy.apk .

# iOS
gogio -target ios -o go_yatzy.app .
```

## Releases

Pre-built binaries for Linux (x64) and Windows (x64) are published automatically via GitHub Actions when a `v*` tag is pushed.

## License

MIT
