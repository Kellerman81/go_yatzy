package main

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// ---- AI speed setting ----

type AISpeed int

const (
	AISpeedSlow   AISpeed = iota // 1800 ms per step
	AISpeedNormal                // 900 ms per step
	AISpeedFast                  // 400 ms per step
)

var aiSpeedDelays = [3]time.Duration{
	1800 * time.Millisecond,
	900 * time.Millisecond,
	400 * time.Millisecond,
}

// aiPhase tracks where the AI is within a single turn
type aiPhase int

const (
	aiPhaseRoll  aiPhase = iota // next: roll the dice
	aiPhaseKeep                 // dice just rolled — next: select which to keep
	aiPhaseScore                // kept dice chosen — next: score a category
)

// ---- colour palette ----

var (
	colBackground  = color.NRGBA{R: 240, G: 240, B: 235, A: 255}
	colHeader      = color.NRGBA{R: 50, G: 80, B: 140, A: 255}
	colHeaderText  = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colSection     = color.NRGBA{R: 200, G: 215, B: 240, A: 255}
	colRowEven     = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colRowOdd      = color.NRGBA{R: 245, G: 248, B: 255, A: 255}
	colCurrentCol  = color.NRGBA{R: 230, G: 245, B: 230, A: 255}
	colDie         = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colDieKept     = color.NRGBA{R: 255, G: 220, B: 50, A: 255}
	colDieDot      = color.NRGBA{R: 30, G: 30, B: 30, A: 255}
	colDieBorder   = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	colTotalRow    = color.NRGBA{R: 50, G: 80, B: 140, A: 40}
	colBonusRow    = color.NRGBA{R: 80, G: 160, B: 80, A: 40}
	colGreen       = color.NRGBA{R: 40, G: 160, B: 40, A: 255}
	colRed         = color.NRGBA{R: 200, G: 50, B: 50, A: 255}
	colGray        = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
	colWinner      = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
	colTransparent = color.NRGBA{A: 0}
)

// ---- UI struct ----

type UI struct {
	theme *material.Theme
	game  *Game
	hs    *HighScores

	// Language
	lang     Lang
	langBtns [2]widget.Clickable

	// Setup screen
	numPlayers      int
	playerCountBtns [MaxPlayers]widget.Clickable
	playerEditors   [MaxPlayers]widget.Editor
	isComputer      [MaxPlayers]bool
	playerTypeBtns  [MaxPlayers]widget.Clickable
	startBtn        widget.Clickable
	editorsInited   bool

	// Game screen
	diceClicks   [NumDice]widget.Clickable
	rollBtn      widget.Clickable
	scoreBtns    [NumCategories][MaxPlayers]widget.Clickable
	scoreList    widget.List

	// AI
	aiActionTime time.Time
	aiCurPhase   aiPhase
	aiSpeed      AISpeed
	aiSpeedBtns  [3]widget.Clickable

	// Game Over
	newGameBtn    widget.Clickable
	highScoresBtn widget.Clickable

	// High Scores
	hsList        widget.List
	statsList     widget.List
	backBtn       widget.Clickable
	hsNewGameBtn  widget.Clickable
	hsTabScores   widget.Clickable // tab button: top scores
	hsTabStats    widget.Clickable // tab button: statistics
	showStatsTab  bool

	// About
	aboutBtn      widget.Clickable
	githubLinkBtn widget.Clickable
	aboutBackBtn  widget.Clickable
}

func (ui *UI) loc() *Locale { return &locales[ui.lang] }

func NewUI(th *material.Theme) *UI {
	ui := &UI{
		theme:      th,
		game:       NewGame(),
		hs:         LoadHighScores(),
		numPlayers: 2,
	}
	ui.scoreList.List.Axis = layout.Vertical
	ui.hsList.List.Axis = layout.Vertical
	ui.statsList.List.Axis = layout.Vertical

	ui.applySettings(LoadSettings())

	return ui
}

func (ui *UI) Run(w *app.Window) error {
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			ui.Layout(gtx)
			e.Frame(&ops)
		}
	}
}

func (ui *UI) Layout(gtx layout.Context) layout.Dimensions {
	// Fill background
	fillRect(gtx.Ops, colBackground, gtx.Constraints.Max)

	// AI turn processing
	if ui.game.State == StatePlaying &&
		ui.game.NumPlayers > 0 &&
		ui.game.Players[ui.game.CurrentPlayer].IsComputer {
		ui.handleAITurn(gtx)
	}

	switch ui.game.State {
	case StateSetup:
		return ui.layoutSetup(gtx)
	case StatePlaying:
		return ui.layoutGame(gtx)
	case StateGameOver:
		return ui.layoutGameOver(gtx)
	case StateHighScores:
		return ui.layoutHighScores(gtx)
	case StateAbout:
		return ui.layoutAbout(gtx)
	}
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

// ---- AI turn ----

func (ui *UI) aiDelay() time.Duration {
	return aiSpeedDelays[ui.aiSpeed]
}

func (ui *UI) handleAITurn(gtx layout.Context) {
	now := gtx.Now
	if now.Before(ui.aiActionTime) {
		gtx.Execute(op.InvalidateCmd{At: ui.aiActionTime})
		return
	}

	g := ui.game
	p := g.Players[g.CurrentPlayer]
	d := ui.aiDelay()

	switch ui.aiCurPhase {

	case aiPhaseRoll:
		// Roll the non-kept dice and pause so the player can see the result
		g.Roll()
		ui.aiCurPhase = aiPhaseKeep
		ui.aiActionTime = now.Add(d)

	case aiPhaseKeep:
		// Decide which dice to keep (highlight them) then pause before next step
		keep := AIDecide(g.Dice.Values, p.Scored)
		g.Dice.Kept = keep

		// Decide whether to roll again
		shouldRollAgain := false
		if g.RollsLeft > 0 {
			bestCat := AIChooseTargetCategory(g.Dice.Values, p.Scored)
			if bestCat >= 0 {
				cur := CalculateScore(bestCat, g.Dice.Values)
				threshold := 35
				if bestCat <= CatSixes {
					threshold = (bestCat + 1) * 3
				}
				shouldRollAgain = cur < threshold
			}
		}

		if shouldRollAgain {
			ui.aiCurPhase = aiPhaseRoll
		} else {
			ui.aiCurPhase = aiPhaseScore
		}
		ui.aiActionTime = now.Add(d)

	case aiPhaseScore:
		cat := AIChooseCategory(g.Dice.Values, p.Scored)
		if cat >= 0 {
			g.ScoreCategory(cat)
			if g.State == StateGameOver {
				ui.saveHighScores()
			}
		}
		// Reset for next turn
		ui.aiCurPhase = aiPhaseRoll
		ui.aiActionTime = now.Add(d / 3)
	}

	gtx.Execute(op.InvalidateCmd{At: ui.aiActionTime})
}

func (ui *UI) saveHighScores() {
	g := ui.game
	winner := g.Winner()
	topScore := 0
	if winner != nil {
		topScore = winner.GrandTotal()
	}
	for _, p := range g.Players {
		if p.IsComputer {
			continue
		}
		entry := HighScoreEntry{
			Name:       p.Name,
			Score:      p.GrandTotal(),
			NumPlayers: g.NumPlayers,
			Date:       time.Now(),
		}
		ui.hs.AddEntry(entry)
		// Win = human has the highest score (ties at the top both count as wins)
		won := p.GrandTotal() == topScore
		ui.hs.UpdateStats(p.Name, p.GrandTotal(), won)
	}
	ui.hs.Save()
}

// ---- Setup screen ----

func (ui *UI) layoutSetup(gtx layout.Context) layout.Dimensions {
	L := ui.loc()
	if !ui.editorsInited {
		for i := 0; i < MaxPlayers; i++ {
			ui.playerEditors[i].SingleLine = true
			ui.playerEditors[i].SetText(fmt.Sprintf(L.DefaultName, i+1))
		}
		ui.editorsInited = true
	}

	th := ui.theme

	return layout.UniformInset(unit.Dp(32)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(gtx,
			// Title
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.H3(th, L.AppTitle)
				lbl.Color = colHeader
				return layout.Center.Layout(gtx, lbl.Layout)
			}),
			layout.Rigid(spacerV(12)),
			// Language selector
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.layoutLangSelector(gtx, th, L)
			}),
			layout.Rigid(spacerV(12)),
			// Number of players
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body1(th, L.LabelNumPlayer).Layout(gtx)
					}),
					layout.Rigid(spacerV(8)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceAround}.Layout(gtx,
							layout.Rigid(ui.numPlayerBtn(gtx, th, 1)),
							layout.Rigid(spacerH(8)),
							layout.Rigid(ui.numPlayerBtn(gtx, th, 2)),
							layout.Rigid(spacerH(8)),
							layout.Rigid(ui.numPlayerBtn(gtx, th, 3)),
							layout.Rigid(spacerH(8)),
							layout.Rigid(ui.numPlayerBtn(gtx, th, 4)),
						)
					}),
				)
			}),
			layout.Rigid(spacerV(12)),
			// Computer speed
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body1(th, L.LabelAISpeed).Layout(gtx)
					}),
					layout.Rigid(spacerV(8)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(ui.aiSpeedBtn(gtx, th, AISpeedSlow)),
							layout.Rigid(spacerH(8)),
							layout.Rigid(ui.aiSpeedBtn(gtx, th, AISpeedNormal)),
							layout.Rigid(spacerH(8)),
							layout.Rigid(ui.aiSpeedBtn(gtx, th, AISpeedFast)),
						)
					}),
				)
			}),
			layout.Rigid(spacerV(12)),
			// Player configuration rows
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				children := make([]layout.FlexChild, 0)
				for i := 0; i < ui.numPlayers; i++ {
					idx := i
					children = append(children,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.layoutPlayerRow(gtx, th, idx, L)
						}),
						layout.Rigid(spacerV(10)),
					)
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			}),
			layout.Rigid(spacerV(20)),
			// Start button
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ui.startBtn.Clicked(gtx) {
					ui.handleStart()
				}
				btn := material.Button(th, &ui.startBtn, L.StartGame)
				btn.Background = colHeader
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
				return layout.Center.Layout(gtx, btn.Layout)
			}),
			// High score button
			layout.Rigid(spacerV(10)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ui.highScoresBtn.Clicked(gtx) {
					ui.game.State = StateHighScores
				}
				btn := material.Button(th, &ui.highScoresBtn, L.Highscores)
				btn.Background = colGray
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
					return btn.Layout(gtx)
				})
			}),
			// About button
			layout.Rigid(spacerV(10)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ui.aboutBtn.Clicked(gtx) {
					ui.game.State = StateAbout
				}
				btn := material.Button(th, &ui.aboutBtn, L.About)
				btn.Background = colGray
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
					return btn.Layout(gtx)
				})
			}),
		)
	})
}

func (ui *UI) layoutLangSelector(gtx layout.Context, th *material.Theme, L *Locale) layout.Dimensions {
	if ui.langBtns[LangDE].Clicked(gtx) {
		ui.lang = LangDE
		ui.editorsInited = false // re-init default names in new language
	}
	if ui.langBtns[LangEN].Clicked(gtx) {
		ui.lang = LangEN
		ui.editorsInited = false
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body1(th, L.LabelLanguage)
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
			return lbl.Layout(gtx)
		}),
		layout.Rigid(spacerH(8)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &ui.langBtns[LangDE], "Deutsch")
			if ui.lang == LangDE {
				btn.Background = colHeader
			} else {
				btn.Background = colGray
			}
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(90))
			return btn.Layout(gtx)
		}),
		layout.Rigid(spacerH(8)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &ui.langBtns[LangEN], "English")
			if ui.lang == LangEN {
				btn.Background = colHeader
			} else {
				btn.Background = colGray
			}
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(90))
			return btn.Layout(gtx)
		}),
	)
}

func (ui *UI) aiSpeedBtn(gtx layout.Context, th *material.Theme, speed AISpeed) layout.Widget {
	L := ui.loc()
	return func(gtx layout.Context) layout.Dimensions {
		if ui.aiSpeedBtns[speed].Clicked(gtx) {
			ui.aiSpeed = speed
		}
		labels := [3]string{L.SpeedSlow, L.SpeedNormal, L.SpeedFast}
		btn := material.Button(th, &ui.aiSpeedBtns[speed], labels[speed])
		if ui.aiSpeed == speed {
			btn.Background = colHeader
		} else {
			btn.Background = colGray
		}
		gtx.Constraints.Min.X = gtx.Dp(unit.Dp(90))
		return btn.Layout(gtx)
	}
}

func (ui *UI) numPlayerBtn(gtx layout.Context, th *material.Theme, n int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if ui.playerCountBtns[n-1].Clicked(gtx) {
			ui.numPlayers = n
		}
		label := strconv.Itoa(n)
		btn := material.Button(th, &ui.playerCountBtns[n-1], label)
		if ui.numPlayers == n {
			btn.Background = colHeader
		} else {
			btn.Background = colGray
		}
		gtx.Constraints.Min.X = gtx.Dp(unit.Dp(60))
		return btn.Layout(gtx)
	}
}

func (ui *UI) layoutPlayerRow(gtx layout.Context, th *material.Theme, idx int, L *Locale) layout.Dimensions {
	if ui.playerTypeBtns[idx].Clicked(gtx) {
		ui.isComputer[idx] = !ui.isComputer[idx]
	}

	typeLabel := L.Human
	btnColor := colGreen
	if ui.isComputer[idx] {
		typeLabel = L.Computer
		btnColor = colRed
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body1(th, fmt.Sprintf(L.PlayerLabel, idx+1))
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(90))
			return lbl.Layout(gtx)
		}),
		layout.Rigid(spacerH(8)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.layoutNameEditor(gtx, th, idx, L)
		}),
		layout.Rigid(spacerH(12)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &ui.playerTypeBtns[idx], typeLabel)
			btn.Background = btnColor
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(110))
			return btn.Layout(gtx)
		}),
	)
}

func (ui *UI) layoutNameEditor(gtx layout.Context, th *material.Theme, idx int, L *Locale) layout.Dimensions {
	const boxW, boxH = 200, 36

	sz := image.Point{X: gtx.Dp(unit.Dp(boxW)), Y: gtx.Dp(unit.Dp(boxH))}
	gtx.Constraints = layout.Exact(sz)

	rr := clip.RRect{
		Rect: image.Rectangle{Max: sz},
		NE: 4, NW: 4, SE: 4, SW: 4,
	}

	// White fill
	paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, rr.Op(gtx.Ops))
	// Border (2 dp)
	paint.FillShape(gtx.Ops, colHeader, clip.Stroke{
		Path:  rr.Path(gtx.Ops),
		Width: float32(gtx.Dp(unit.Dp(2))),
	}.Op())

	// Editor with inset so text doesn't touch the border
	return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			ed := material.Editor(th, &ui.playerEditors[idx], L.NameHint)
			ed.Color = color.NRGBA{A: 255}
			ed.HintColor = colGray
			return ed.Layout(gtx)
		},
	)
}

func (ui *UI) handleStart() {
	L := ui.loc()
	players := make([]*Player, ui.numPlayers)
	setup := make([]PlayerSetup, MaxPlayers)
	for i := 0; i < MaxPlayers; i++ {
		name := ui.playerEditors[i].Text()
		if name == "" {
			name = fmt.Sprintf(L.DefaultName, i+1)
		}
		setup[i] = PlayerSetup{Name: name, IsComputer: ui.isComputer[i]}
		if i < ui.numPlayers {
			players[i] = &Player{Name: name, IsComputer: ui.isComputer[i]}
		}
	}
	SaveSettings(&Settings{
		NumPlayers: ui.numPlayers,
		Players:    setup,
		Lang:       ui.lang,
		AISpeed:    ui.aiSpeed,
	})
	ui.game.StartGame(players)
	ui.aiCurPhase = aiPhaseRoll
	ui.aiActionTime = time.Now().Add(ui.aiDelay())
}

// ---- Game screen ----

func (ui *UI) layoutGame(gtx layout.Context) layout.Dimensions {
	g := ui.game
	th := ui.theme
	p := g.Players[g.CurrentPlayer]

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Top bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.layoutTopBar(gtx, th, p)
		}),
		// Dice area
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(11), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.layoutDiceArea(gtx, th)
			})
		}),
		// Score sheet fills remaining space
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.layoutScoreSheet(gtx, th)
			})
		}),
	)
}

func (ui *UI) layoutTopBar(gtx layout.Context, th *material.Theme, p *Player) layout.Dimensions {
	L := ui.loc()
	barH := gtx.Dp(unit.Dp(48))
	gtx.Constraints = layout.Exact(image.Point{X: gtx.Constraints.Max.X, Y: barH})
	fillRect(gtx.Ops, colHeader, image.Point{X: gtx.Constraints.Max.X, Y: barH})
	return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.H6(th, L.AppTitle)
					lbl.Color = colHeaderText
					return lbl.Layout(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					name := p.Name
					if p.IsComputer {
						name += L.ComputerSuffix
					}
					lbl := material.Body1(th, fmt.Sprintf(L.CurrentPlayer, name))
					lbl.Color = colHeaderText
					return layout.Center.Layout(gtx, lbl.Layout)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body1(th, fmt.Sprintf(L.RollsLeft, ui.game.RollsLeft))
					lbl.Color = colHeaderText
					return lbl.Layout(gtx)
				}),
			)
		},
	)
}

func (ui *UI) layoutDiceArea(gtx layout.Context, th *material.Theme) layout.Dimensions {
	L := ui.loc()
	g := ui.game
	isHuman := !g.Players[g.CurrentPlayer].IsComputer

	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		// Dice row
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceAround, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, NumDice*2-1)
					for i := 0; i < NumDice; i++ {
						idx := i
						children[i*2] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.layoutDie(gtx, idx, isHuman)
						})
						if i < NumDice-1 {
							children[i*2+1] = layout.Rigid(spacerH(12))
						}
					}
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
				}),
			)
		}),
		layout.Rigid(spacerV(10)),
		// Roll button
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !isHuman {
				lbl := material.Body2(th, L.ComputerPlaying)
				lbl.Color = colGray
				return layout.Center.Layout(gtx, lbl.Layout)
			}
			canRoll := g.RollsLeft > 0
			if ui.rollBtn.Clicked(gtx) && canRoll {
				g.Roll()
			}
			btn := material.Button(th, &ui.rollBtn, fmt.Sprintf(L.RollBtn, g.RollsLeft))
			if !canRoll {
				btn.Background = colGray
			} else {
				btn.Background = colHeader
			}
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
			return layout.Center.Layout(gtx, btn.Layout)
		}),
	)
}

func (ui *UI) layoutDie(gtx layout.Context, idx int, interactive bool) layout.Dimensions {
	g := ui.game
	dieSize := unit.Dp(62)
	sz := gtx.Dp(dieSize)

	if interactive && g.HasRolled {
		if ui.diceClicks[idx].Clicked(gtx) {
			g.ToggleKeep(idx)
		}
	}

	drawFn := func(gtx layout.Context) layout.Dimensions {
		kept := g.Dice.Kept[idx]
		val := g.Dice.Values[idx]

		bgColor := colDie
		if kept {
			bgColor = colDieKept
		}

		// Draw die background (rounded rect)
		rrect := clip.RRect{
			Rect: image.Rectangle{Max: image.Point{X: sz, Y: sz}},
			NE: 8, NW: 8, SE: 8, SW: 8,
		}
		paint.FillShape(gtx.Ops, bgColor, rrect.Op(gtx.Ops))
		// Border
		paint.FillShape(gtx.Ops, colDieBorder, clip.Stroke{
			Path:  rrect.Path(gtx.Ops),
			Width: float32(gtx.Dp(unit.Dp(2))),
		}.Op())

		// Draw dots
		if val >= 1 && val <= 6 {
			drawDots(gtx.Ops, val, sz)
		}

		return layout.Dimensions{Size: image.Point{X: sz, Y: sz}}
	}

	if interactive && g.HasRolled {
		return ui.diceClicks[idx].Layout(gtx, drawFn)
	}
	return drawFn(gtx)
}

// dot positions as (x, y) fractions of die size, for each face value
var dotPositions = [7][][2]float32{
	{},
	{{0.5, 0.5}},
	{{0.25, 0.25}, {0.75, 0.75}},
	{{0.25, 0.25}, {0.5, 0.5}, {0.75, 0.75}},
	{{0.25, 0.25}, {0.75, 0.25}, {0.25, 0.75}, {0.75, 0.75}},
	{{0.25, 0.25}, {0.75, 0.25}, {0.5, 0.5}, {0.25, 0.75}, {0.75, 0.75}},
	{{0.25, 0.25}, {0.75, 0.25}, {0.25, 0.5}, {0.75, 0.5}, {0.25, 0.75}, {0.75, 0.75}},
}

func drawDots(ops *op.Ops, val, sz int) {
	r := sz / 9 // dot radius
	for _, pos := range dotPositions[val] {
		cx := int(float32(sz) * pos[0])
		cy := int(float32(sz) * pos[1])
		paint.FillShape(ops, colDieDot,
			clip.Ellipse{
				Min: image.Point{X: cx - r, Y: cy - r},
				Max: image.Point{X: cx + r, Y: cy + r},
			}.Op(ops))
	}
}

// ---- Score sheet ----

// Score sheet row definitions
type rowKind int

const (
	rowHeader  rowKind = iota
	rowSection         // section divider
	rowCat             // scoring category
	rowBonus           // upper section bonus
	rowSubtotal        // subtotal row
	rowTotal           // grand total
)

type sheetRow struct {
	kind   rowKind
	label  string
	catIdx int // -1 for non-category rows
}

var scoreSheetRows = []sheetRow{
	{rowSection, "upper", -1},
	{rowCat, "", CatOnes},
	{rowCat, "", CatTwos},
	{rowCat, "", CatThrees},
	{rowCat, "", CatFours},
	{rowCat, "", CatFives},
	{rowCat, "", CatSixes},
	{rowSubtotal, "", -1},
	{rowBonus, "", -1},
	{rowSection, "lower", -1},
	{rowCat, "", CatThreeOfKind},
	{rowCat, "", CatFourOfKind},
	{rowCat, "", CatFullHouse},
	{rowCat, "", CatSmallStraight},
	{rowCat, "", CatLargeStraight},
	{rowCat, "", CatKniffel},
	{rowCat, "", CatChance},
	{rowTotal, "", -1},
}

const (
	colNameW  = 170
	colScoreW = 100
	rowH      = 30
	headerH   = 32
)

func (ui *UI) layoutScoreSheet(gtx layout.Context, th *material.Theme) layout.Dimensions {
	g := ui.game
	numCols := 1 + g.NumPlayers // name col + one per player

	// Header
	headerDims := ui.layoutSheetHeader(gtx, th)
	_ = headerDims

	totalW := colNameW + colScoreW*g.NumPlayers
	_ = totalW
	_ = numCols

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.layoutSheetHeader(gtx, th)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(th, &ui.scoreList).Layout(gtx, len(scoreSheetRows), func(gtx layout.Context, i int) layout.Dimensions {
				return ui.layoutSheetRow(gtx, th, i)
			})
		}),
	)
}

func (ui *UI) layoutSheetHeader(gtx layout.Context, th *material.Theme) layout.Dimensions {
	L := ui.loc()
	g := ui.game
	h := gtx.Dp(unit.Dp(headerH))
	w := gtx.Dp(unit.Dp(colNameW + colScoreW*g.NumPlayers))
	fillRect(gtx.Ops, colHeader, image.Point{X: w, Y: h})

	children := make([]layout.FlexChild, 1+g.NumPlayers)
	children[0] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return fixedCell(gtx, th, L.ColCategory, colNameW, headerH, colHeaderText, true)
	})
	for i := 0; i < g.NumPlayers; i++ {
		pi := i
		marker := ""
		if pi == g.CurrentPlayer {
			marker = " ▶"
		}
		name := g.Players[pi].Name + marker
		children[1+pi] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedCell(gtx, th, name, colScoreW, headerH, colHeaderText, true)
		})
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

func (ui *UI) layoutSheetRow(gtx layout.Context, th *material.Theme, rowIdx int) layout.Dimensions {
	L := ui.loc()
	g := ui.game
	row := scoreSheetRows[rowIdx]

	// Background colour for the row
	bgCol := colRowEven
	if rowIdx%2 == 1 {
		bgCol = colRowOdd
	}

	switch row.kind {
	case rowSection:
		bgCol = colSection
	case rowBonus:
		bgCol = colBonusRow
	case rowSubtotal:
		bgCol = colBonusRow
	case rowTotal:
		bgCol = colTotalRow
	}

	h := gtx.Dp(unit.Dp(rowH))
	w := gtx.Dp(unit.Dp(colNameW + colScoreW*g.NumPlayers))
	fillRect(gtx.Ops, bgCol, image.Point{X: w, Y: h})

	// Label for category rows — resolved from active locale
	var label string
	switch row.kind {
	case rowCat:
		label = L.Categories[row.catIdx]
	case rowSection:
		if row.label == "upper" {
			label = L.UpperSection
		} else {
			label = L.LowerSection
		}
	case rowSubtotal:
		label = L.Subtotal
	case rowBonus:
		label = L.BonusRow
	case rowTotal:
		label = L.GrandTotal
	}

	children := make([]layout.FlexChild, 1+g.NumPlayers)
	children[0] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		bold := row.kind == rowSection || row.kind == rowTotal || row.kind == rowSubtotal
		return fixedCell(gtx, th, label, colNameW, rowH, color.NRGBA{A: 255}, bold)
	})

	for i := 0; i < g.NumPlayers; i++ {
		pi := i
		children[1+pi] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.layoutScoreCell(gtx, th, row, pi)
		})
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

func (ui *UI) layoutScoreCell(gtx layout.Context, th *material.Theme, row sheetRow, playerIdx int) layout.Dimensions {
	g := ui.game
	p := g.Players[playerIdx]
	isCurrentPlayer := playerIdx == g.CurrentPlayer

	cellW := unit.Dp(colScoreW)
	cellH := unit.Dp(rowH)

	// Highlight current player's column for category rows
	if isCurrentPlayer && row.kind == rowCat {
		fillRect(gtx.Ops, colCurrentCol, image.Point{
			X: gtx.Dp(cellW),
			Y: gtx.Dp(cellH),
		})
	}

	switch row.kind {
	case rowSection:
		return layout.Dimensions{Size: image.Point{X: gtx.Dp(cellW), Y: gtx.Dp(cellH)}}

	case rowBonus:
		bonus := p.UpperBonus()
		upper := p.TotalUpper()
		var text string
		if bonus > 0 {
			text = "+35"
		} else {
			text = fmt.Sprintf("%d/63", upper)
		}
		col := color.NRGBA{A: 255}
		if bonus > 0 {
			col = colGreen
		}
		return fixedCell(gtx, th, text, colScoreW, rowH, col, true)

	case rowSubtotal:
		return fixedCell(gtx, th, strconv.Itoa(p.TotalUpper()), colScoreW, rowH, color.NRGBA{A: 255}, true)

	case rowTotal:
		return fixedCell(gtx, th, strconv.Itoa(p.GrandTotal()), colScoreW, rowH, colHeader, true)

	case rowCat:
		cat := row.catIdx
		if p.Scored[cat] {
			// Already scored – show locked value
			s := strconv.Itoa(p.Scores[cat])
			return fixedCell(gtx, th, s, colScoreW, rowH, color.NRGBA{A: 255}, false)
		}

		if !isCurrentPlayer || !g.HasRolled {
			// Empty for other players or before first roll
			return layout.Dimensions{Size: image.Point{X: gtx.Dp(cellW), Y: gtx.Dp(cellH)}}
		}

		// Current player, not yet scored: show clickable potential score
		potential := CalculateScore(cat, g.Dice.Values)
		label := strconv.Itoa(potential)
		if potential == 0 {
			label = "0 ✗"
		}

		if ui.scoreBtns[cat][playerIdx].Clicked(gtx) {
			g.ScoreCategory(cat)
			if g.State == StateGameOver {
				ui.saveHighScores()
			}
		}

		scoreColor := colGreen
		if potential == 0 {
			scoreColor = colRed
		}
		sz := image.Point{X: gtx.Dp(cellW), Y: gtx.Dp(cellH)}
		gtx.Constraints = layout.Exact(sz)
		return ui.scoreBtns[cat][playerIdx].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, scoreColor, clip.Rect{Max: sz}.Op())
			lbl := material.Body2(th, label)
			lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			return layout.Center.Layout(gtx, lbl.Layout)
		})
	}

	return layout.Dimensions{Size: image.Point{X: gtx.Dp(cellW), Y: gtx.Dp(cellH)}}
}

// ---- Game Over screen ----

func (ui *UI) layoutGameOver(gtx layout.Context) layout.Dimensions {
	L := ui.loc()
	th := ui.theme
	g := ui.game
	winner := g.Winner()

	return layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.H3(th, L.GameOverTitle)
				lbl.Color = colHeader
				return layout.Center.Layout(gtx, lbl.Layout)
			}),
			layout.Rigid(spacerV(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				winText := ""
				if winner != nil {
					if g.NumPlayers == 1 {
						winText = fmt.Sprintf(L.ResultSingle, winner.GrandTotal())
					} else {
						winText = fmt.Sprintf(L.ResultWinner, winner.Name, winner.GrandTotal())
					}
				}
				lbl := material.H5(th, winText)
				lbl.Color = color.NRGBA{R: 180, G: 120, B: 0, A: 255}
				return layout.Center.Layout(gtx, lbl.Layout)
			}),
			layout.Rigid(spacerV(16)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.layoutFinalScoresTable(gtx, th)
			}),
			layout.Rigid(spacerV(16)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceAround}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if ui.newGameBtn.Clicked(gtx) {
							ui.resetGame()
						}
						btn := material.Button(th, &ui.newGameBtn, L.NewGame)
						btn.Background = colHeader
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
						return btn.Layout(gtx)
					}),
					layout.Rigid(spacerH(16)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if ui.highScoresBtn.Clicked(gtx) {
							ui.game.State = StateHighScores
						}
						btn := material.Button(th, &ui.highScoresBtn, L.Highscores)
						btn.Background = colGray
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
						return btn.Layout(gtx)
					}),
				)
			}),
		)
	})
}

func (ui *UI) layoutFinalScoresTable(gtx layout.Context, th *material.Theme) layout.Dimensions {
	L := ui.loc()
	g := ui.game
	winner := g.Winner()

	rows := make([]layout.FlexChild, 2)

	// Header
	rows[0] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		hdr := make([]layout.FlexChild, 1+g.NumPlayers)
		hdr[0] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedCell(gtx, th, L.ColPlayers, 200, 36, colHeaderText, true)
		})
		for i, p := range g.Players {
			pi := i
			_ = p
			hdr[1+pi] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return fixedCell(gtx, th, g.Players[pi].Name, 120, 36, colHeaderText, true)
			})
		}
		w := gtx.Dp(unit.Dp(200 + 120*g.NumPlayers))
		fillRect(gtx.Ops, colHeader, image.Point{X: w, Y: gtx.Dp(unit.Dp(36))})
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, hdr...)
	})

	// Score rows for each category
	scoreRowLabels := []struct {
		label string
		fn    func(*Player) string
	}{
		{L.FinalUpper, func(p *Player) string { return strconv.Itoa(p.TotalUpper()) }},
		{L.FinalBonus, func(p *Player) string {
			if p.UpperBonus() > 0 {
				return "+35"
			}
			return "0"
		}},
		{L.FinalLower, func(p *Player) string { return strconv.Itoa(p.TotalLower()) }},
		{L.FinalTotal, func(p *Player) string { return strconv.Itoa(p.GrandTotal()) }},
	}

	tableRows := make([]layout.FlexChild, len(scoreRowLabels))
	for ri, sr := range scoreRowLabels {
		ri := ri
		sr := sr
		isTotal := ri == len(scoreRowLabels)-1
		tableRows[ri] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			bgC := colRowEven
			if ri%2 == 1 {
				bgC = colRowOdd
			}
			if isTotal {
				bgC = colTotalRow
			}
			w := gtx.Dp(unit.Dp(200 + 120*g.NumPlayers))
			fillRect(gtx.Ops, bgC, image.Point{X: w, Y: gtx.Dp(unit.Dp(32))})

			cells := make([]layout.FlexChild, 1+g.NumPlayers)
			cells[0] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return fixedCell(gtx, th, sr.label, 200, 32, color.NRGBA{A: 255}, isTotal)
			})
			for pi, p := range g.Players {
				pi := pi
				p := p
				textC := color.NRGBA{A: 255}
				if isTotal && p == winner {
					textC = color.NRGBA{R: 180, G: 120, B: 0, A: 255}
				}
				cells[1+pi] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, sr.fn(p), 120, 32, textC, isTotal)
				})
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, cells...)
		})
	}

	rows[1] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, tableRows...)
	})

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func (ui *UI) resetGame() {
	ui.game = NewGame()
	ui.aiActionTime = time.Time{}
	ui.applySettings(LoadSettings())
}

// applySettings restores player setup from saved settings.
// If s is nil the editors are left uninitialised so layoutSetup fills them with locale defaults.
func (ui *UI) applySettings(s *Settings) {
	if s == nil {
		ui.editorsInited = false
		return
	}
	if s.NumPlayers >= 1 && s.NumPlayers <= MaxPlayers {
		ui.numPlayers = s.NumPlayers
	}
	for i, p := range s.Players {
		if i < MaxPlayers {
			ui.playerEditors[i].SingleLine = true
			if p.Name != "" {
				ui.playerEditors[i].SetText(p.Name)
			}
			ui.isComputer[i] = p.IsComputer
		}
	}
	ui.lang = s.Lang
	ui.aiSpeed = s.AISpeed
	ui.editorsInited = true
}

// ---- High Scores screen ----

func (ui *UI) layoutHighScores(gtx layout.Context) layout.Dimensions {
	L := ui.loc()
	th := ui.theme

	return layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.H3(th, L.Highscores)
				lbl.Color = colHeader
				return layout.Center.Layout(gtx, lbl.Layout)
			}),
			layout.Rigid(spacerV(12)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.layoutHSTabs(gtx, th, L)
			}),
			layout.Rigid(spacerV(8)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if ui.showStatsTab {
					return ui.layoutStatsTable(gtx, th, L)
				}
				return ui.layoutTopScoresTable(gtx, th, L)
			}),
			layout.Rigid(spacerV(12)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceAround}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if ui.backBtn.Clicked(gtx) {
							ui.game.State = StateSetup
						}
						btn := material.Button(th, &ui.backBtn, L.Back)
						btn.Background = colGray
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(140))
						return btn.Layout(gtx)
					}),
					layout.Rigid(spacerH(16)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if ui.hsNewGameBtn.Clicked(gtx) {
							ui.resetGame()
						}
						btn := material.Button(th, &ui.hsNewGameBtn, L.NewGame)
						btn.Background = colHeader
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(140))
						return btn.Layout(gtx)
					}),
				)
			}),
		)
	})
}

func (ui *UI) layoutHSTabs(gtx layout.Context, th *material.Theme, L *Locale) layout.Dimensions {
	if ui.hsTabScores.Clicked(gtx) {
		ui.showStatsTab = false
	}
	if ui.hsTabStats.Clicked(gtx) {
		ui.showStatsTab = true
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &ui.hsTabScores, L.TabTopScores)
			if !ui.showStatsTab {
				btn.Background = colHeader
			} else {
				btn.Background = colGray
			}
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
			return btn.Layout(gtx)
		}),
		layout.Rigid(spacerH(8)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &ui.hsTabStats, L.TabStats)
			if ui.showStatsTab {
				btn.Background = colHeader
			} else {
				btn.Background = colGray
			}
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
			return btn.Layout(gtx)
		}),
	)
}

func (ui *UI) layoutTopScoresTable(gtx layout.Context, th *material.Theme, L *Locale) layout.Dimensions {
	hs := ui.hs
	const tableW = 560
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			fillRect(gtx.Ops, colHeader, image.Point{X: gtx.Dp(unit.Dp(tableW)), Y: gtx.Dp(unit.Dp(36))})
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColRank, 40, 36, colHeaderText, true)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColName, 200, 36, colHeaderText, true)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColScore, 100, 36, colHeaderText, true)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColPlayers, 80, 36, colHeaderText, true)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColDate, 140, 36, colHeaderText, true)
				}),
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(hs.Entries) == 0 {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body1(th, L.NoEntries)
					lbl.Color = colGray
					return lbl.Layout(gtx)
				})
			}
			return material.List(th, &ui.hsList).Layout(gtx, len(hs.Entries), func(gtx layout.Context, i int) layout.Dimensions {
				e := hs.Entries[i]
				bgC := colRowEven
				if i%2 == 1 {
					bgC = colRowOdd
				}
				if i == 0 {
					bgC = colWinner
				}
				fillRect(gtx.Ops, bgC, image.Point{X: gtx.Dp(unit.Dp(tableW)), Y: gtx.Dp(unit.Dp(32))})
				rank := strconv.Itoa(i + 1)
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, rank, 40, 32, color.NRGBA{A: 255}, i == 0)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, e.Name, 200, 32, color.NRGBA{A: 255}, i == 0)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, strconv.Itoa(e.Score), 100, 32, color.NRGBA{A: 255}, i == 0)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, strconv.Itoa(e.NumPlayers), 80, 32, color.NRGBA{A: 255}, false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, e.Date.Format("02.01.2006"), 140, 32, color.NRGBA{A: 255}, false)
					}),
				)
			})
		}),
	)
}

func (ui *UI) layoutStatsTable(gtx layout.Context, th *material.Theme, L *Locale) layout.Dimensions {
	hs := ui.hs
	stats := hs.SortedStats()
	const tableW = 620
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			fillRect(gtx.Ops, colHeader, image.Point{X: gtx.Dp(unit.Dp(tableW)), Y: gtx.Dp(unit.Dp(36))})
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColName, 200, 36, colHeaderText, true)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColGames, 80, 36, colHeaderText, true)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColWins, 80, 36, colHeaderText, true)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColLosses, 100, 36, colHeaderText, true)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColWinRate, 80, 36, colHeaderText, true)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedCell(gtx, th, L.ColBestScore, 100, 36, colHeaderText, true)
				}),
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(stats) == 0 {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body1(th, L.NoStats)
					lbl.Color = colGray
					return lbl.Layout(gtx)
				})
			}
			return material.List(th, &ui.statsList).Layout(gtx, len(stats), func(gtx layout.Context, i int) layout.Dimensions {
				row := stats[i]
				s := row.Stats
				bgC := colRowEven
				if i%2 == 1 {
					bgC = colRowOdd
				}
				fillRect(gtx.Ops, bgC, image.Point{X: gtx.Dp(unit.Dp(tableW)), Y: gtx.Dp(unit.Dp(32))})
				winRateStr := fmt.Sprintf("%.0f%%", s.WinRate())
				winsCol := color.NRGBA{A: 255}
				if s.Wins > 0 {
					winsCol = colGreen
				}
				lossCol := color.NRGBA{A: 255}
				if s.Losses > 0 {
					lossCol = colRed
				}
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, row.Name, 200, 32, color.NRGBA{A: 255}, true)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, strconv.Itoa(s.Games()), 80, 32, color.NRGBA{A: 255}, false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, strconv.Itoa(s.Wins), 80, 32, winsCol, false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, strconv.Itoa(s.Losses), 100, 32, lossCol, false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, winRateStr, 80, 32, color.NRGBA{A: 255}, false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedCell(gtx, th, strconv.Itoa(s.BestScore), 100, 32, color.NRGBA{A: 255}, false)
					}),
				)
			})
		}),
	)
}

// ---- About screen ----

func (ui *UI) layoutAbout(gtx layout.Context) layout.Dimensions {
	L := ui.loc()
	th := ui.theme

	if ui.aboutBackBtn.Clicked(gtx) {
		ui.game.State = StateSetup
	}
	if ui.githubLinkBtn.Clicked(gtx) {
		openURL("https://" + L.AboutGitHub)
	}

	return layout.UniformInset(unit.Dp(32)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Title
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.H4(th, L.AboutTitle)
				lbl.Color = colHeader
				return layout.Center.Layout(gtx, lbl.Layout)
			}),
			layout.Rigid(spacerV(24)),
			// Description
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, L.AboutDesc)
				lbl.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
				return lbl.Layout(gtx)
			}),
			layout.Rigid(spacerV(24)),
			// Source code label + clickable link
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body1(th, L.AboutSourceCode+" ")
						lbl.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.githubLinkBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(th, L.AboutGitHub)
							lbl.Color = color.NRGBA{R: 0, G: 80, B: 200, A: 255}
							dims := lbl.Layout(gtx)
							// Draw underline
							lineY := float32(dims.Size.Y) - float32(gtx.Dp(unit.Dp(2)))
							paint.FillShape(gtx.Ops,
								color.NRGBA{R: 0, G: 80, B: 200, A: 255},
								clip.Rect{
									Min: image.Point{X: 0, Y: int(lineY)},
									Max: image.Point{X: dims.Size.X, Y: int(lineY) + gtx.Dp(unit.Dp(1))},
								}.Op(),
							)
							return dims
						})
					}),
				)
			}),
			layout.Rigid(spacerV(16)),
			// License
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(th, L.AboutLicense)
				lbl.Color = colGray
				return lbl.Layout(gtx)
			}),
			layout.Rigid(spacerV(32)),
			// Back button
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(th, &ui.aboutBackBtn, L.Back)
				btn.Background = colGray
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
					return btn.Layout(gtx)
				})
			}),
		)
	})
}

// ---- Helpers ----

func fixedCell(gtx layout.Context, th *material.Theme, text string, w, h int, col color.NRGBA, bold bool) layout.Dimensions {
	sz := image.Point{X: gtx.Dp(unit.Dp(w)), Y: gtx.Dp(unit.Dp(h))}
	gtx.Constraints = layout.Exact(sz)
	return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		var lbl material.LabelStyle
		if bold {
			lbl = material.Body1(th, text)
			lbl.Font.Weight = 700
		} else {
			lbl = material.Body2(th, text)
		}
		lbl.Color = col
		lbl.MaxLines = 1
		return lbl.Layout(gtx)
	})
}

func fillRect(ops *op.Ops, col color.NRGBA, sz image.Point) {
	paint.FillShape(ops, col, clip.Rect{Max: sz}.Op())
}

func spacerV(dp unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Point{Y: gtx.Dp(dp)}}
	}
}

func spacerH(dp unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Point{X: gtx.Dp(dp)}}
	}
}
