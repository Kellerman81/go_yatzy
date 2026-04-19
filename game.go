package main

import "math/rand"

const (
	NumDice       = 5
	MaxRolls      = 3
	NumCategories = 13
	MaxPlayers    = 4
)

// Score category indices
const (
	CatOnes = iota
	CatTwos
	CatThrees
	CatFours
	CatFives
	CatSixes
	CatThreeOfKind
	CatFourOfKind
	CatFullHouse
	CatSmallStraight
	CatLargeStraight
	CatKniffel
	CatChance
)

// CategoryNames is kept for fallback; the active locale's Categories is used in the UI.
var CategoryNames = [NumCategories]string{
	"Ones", "Twos", "Threes", "Fours", "Fives", "Sixes",
	"Three of a Kind", "Four of a Kind", "Full House",
	"Small Straight", "Large Straight", "Yatzy", "Chance",
}

// GameState represents the current phase of the application
type GameState int

const (
	StateSetup     GameState = iota
	StatePlaying
	StateGameOver
	StateHighScores
	StateAbout
)

// Dice holds the 5 dice and their keep-flags
type Dice struct {
	Values [NumDice]int
	Kept   [NumDice]bool
}

func (d *Dice) Roll(rng *rand.Rand) {
	for i := range d.Values {
		if !d.Kept[i] {
			d.Values[i] = rng.Intn(6) + 1
		}
	}
}

func (d *Dice) ReleaseAll() {
	for i := range d.Kept {
		d.Kept[i] = false
	}
}

// Player holds all state for a single player
type Player struct {
	Name       string
	IsComputer bool
	Scores     [NumCategories]int
	Scored     [NumCategories]bool
}

func (p *Player) TotalUpper() int {
	total := 0
	for i := CatOnes; i <= CatSixes; i++ {
		if p.Scored[i] {
			total += p.Scores[i]
		}
	}
	return total
}

func (p *Player) UpperBonus() int {
	if p.TotalUpper() >= 63 {
		return 35
	}
	return 0
}

func (p *Player) TotalLower() int {
	total := 0
	for i := CatThreeOfKind; i <= CatChance; i++ {
		if p.Scored[i] {
			total += p.Scores[i]
		}
	}
	return total
}

func (p *Player) GrandTotal() int {
	return p.TotalUpper() + p.UpperBonus() + p.TotalLower()
}

func (p *Player) AllScored() bool {
	for _, s := range p.Scored {
		if !s {
			return false
		}
	}
	return true
}

// Game is the central game state
type Game struct {
	State         GameState
	Players       []*Player
	NumPlayers    int
	CurrentPlayer int
	Dice          Dice
	RollsLeft     int
	HasRolled     bool
	Rng           *rand.Rand
}

func NewGame() *Game {
	return &Game{
		State: StateSetup,
		Rng:   rand.New(rand.NewSource(rand.Int63())),
	}
}

func (g *Game) StartGame(players []*Player) {
	g.Players = players
	g.NumPlayers = len(players)
	g.CurrentPlayer = 0
	g.RollsLeft = MaxRolls
	g.HasRolled = false
	g.Dice.ReleaseAll()
	g.State = StatePlaying
}

func (g *Game) Roll() {
	if g.RollsLeft <= 0 || g.State != StatePlaying {
		return
	}
	g.Dice.Roll(g.Rng)
	g.RollsLeft--
	g.HasRolled = true
}

func (g *Game) ToggleKeep(idx int) {
	if !g.HasRolled || g.State != StatePlaying {
		return
	}
	g.Dice.Kept[idx] = !g.Dice.Kept[idx]
}

func (g *Game) ScoreCategory(cat int) bool {
	if !g.HasRolled || g.State != StatePlaying {
		return false
	}
	p := g.Players[g.CurrentPlayer]
	if p.Scored[cat] {
		return false
	}
	p.Scores[cat] = CalculateScore(cat, g.Dice.Values)
	p.Scored[cat] = true
	g.nextTurn()
	return true
}

func (g *Game) nextTurn() {
	g.CurrentPlayer = (g.CurrentPlayer + 1) % g.NumPlayers

	// Check if all players have scored all categories
	allDone := true
	for _, p := range g.Players {
		if !p.AllScored() {
			allDone = false
			break
		}
	}
	if allDone {
		g.State = StateGameOver
		return
	}

	g.RollsLeft = MaxRolls
	g.HasRolled = false
	g.Dice.ReleaseAll()
}

func (g *Game) Winner() *Player {
	if len(g.Players) == 0 {
		return nil
	}
	winner := g.Players[0]
	for _, p := range g.Players[1:] {
		if p.GrandTotal() > winner.GrandTotal() {
			winner = p
		}
	}
	return winner
}

// ---- Scoring functions ----

func CalculateScore(cat int, dice [NumDice]int) int {
	counts := countDice(dice)
	sum := sumDice(dice)

	switch cat {
	case CatOnes:
		return counts[1] * 1
	case CatTwos:
		return counts[2] * 2
	case CatThrees:
		return counts[3] * 3
	case CatFours:
		return counts[4] * 4
	case CatFives:
		return counts[5] * 5
	case CatSixes:
		return counts[6] * 6
	case CatThreeOfKind:
		for _, c := range counts {
			if c >= 3 {
				return sum
			}
		}
		return 0
	case CatFourOfKind:
		for _, c := range counts {
			if c >= 4 {
				return sum
			}
		}
		return 0
	case CatFullHouse:
		hasThree, hasTwo := false, false
		for _, c := range counts {
			if c == 3 {
				hasThree = true
			} else if c == 2 {
				hasTwo = true
			}
		}
		if hasThree && hasTwo {
			return 25
		}
		return 0
	case CatSmallStraight:
		if hasSequence(counts, 4) {
			return 30
		}
		return 0
	case CatLargeStraight:
		if hasSequence(counts, 5) {
			return 40
		}
		return 0
	case CatKniffel:
		for _, c := range counts {
			if c == 5 {
				return 50
			}
		}
		return 0
	case CatChance:
		return sum
	}
	return 0
}

func countDice(dice [NumDice]int) [7]int {
	var counts [7]int
	for _, v := range dice {
		if v >= 1 && v <= 6 {
			counts[v]++
		}
	}
	return counts
}

func sumDice(dice [NumDice]int) int {
	sum := 0
	for _, v := range dice {
		sum += v
	}
	return sum
}

func hasSequence(counts [7]int, length int) bool {
	consecutive := 0
	for i := 1; i <= 6; i++ {
		if counts[i] > 0 {
			consecutive++
			if consecutive >= length {
				return true
			}
		} else {
			consecutive = 0
		}
	}
	return false
}
