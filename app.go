package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Tab identifies which mode is active.
type Tab int

const (
	TabSim Tab = iota
	TabPattern
	TabHeatmap
	tabCount
)

var tabNames = [tabCount]string{"Simulator", "Pattern", "Heatmap"}

const tabBarH = 30

// App is the top-level ebiten.Game that wraps all three modes with a tab bar.
type App struct {
	activeTab Tab
	sim       *Game
	heatmap   *HeatmapGame
	pattern   *PatternGame
}

func NewApp() *App {
	return &App{
		sim:     NewGame(),
		heatmap: NewHeatmapGame(),
		pattern: &PatternGame{},
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenW, ScreenH
}

func (a *App) Update() error {
	// Tab click detection
	btnW := ScreenW / int(tabCount)
	for i := 0; i < int(tabCount); i++ {
		if btnClicked(i*btnW, 0, btnW, tabBarH) {
			a.activeTab = Tab(i)
			return nil // consume click, don't pass to sub-game
		}
	}

	switch a.activeTab {
	case TabSim:
		return a.sim.Update()
	case TabHeatmap:
		return a.heatmap.Update()
	case TabPattern:
		return a.pattern.Update()
	}
	return nil
}

func (a *App) Draw(screen *ebiten.Image) {
	// Tab bar
	btnW := ScreenW / int(tabCount)
	for i := 0; i < int(tabCount); i++ {
		drawButton(screen, i*btnW, 0, btnW, tabBarH, tabNames[i], Tab(i) == a.activeTab)
	}

	// Active tab content
	switch a.activeTab {
	case TabSim:
		a.sim.Draw(screen)
	case TabHeatmap:
		a.heatmap.Draw(screen)
	case TabPattern:
		a.pattern.Draw(screen)
	}
}
