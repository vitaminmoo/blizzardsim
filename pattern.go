package main

import (
	"fmt"
	"image/color"
	"math"

	"blizzardsim/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type PatternGame struct {
	castX, castY int // current cast position in subtiles
	valid        bool
}

func (g *PatternGame) Update() error {
	mx, my := ebiten.CursorPosition()
	// Prefer active touch position (for mobile/touchscreen)
	if tids := ebiten.AppendTouchIDs(nil); len(tids) > 0 {
		mx, my = ebiten.TouchPosition(tids[0])
	}

	gx, gy := screenToIso(mx, my)
	sx := int(math.Floor(gx))
	sy := int(math.Floor(gy))
	if sx >= 0 && sx < sim.FieldW && sy >= 0 && sy < sim.FieldH {
		g.valid = true
		g.castX = sx
		g.castY = sy
	}
	// Keep showing last valid pattern when touch ends / cursor leaves grid
	return nil
}

func (g *PatternGame) Draw(screen *ebiten.Image) {
	drawIsometricGrid(screen)

	if !g.valid {
		ebitenutil.DebugPrintAt(screen, "Touch or move mouse over the grid", 10, tabBarH+2)
		return
	}

	// Compute all 25 shard positions for this cast
	type shardPos struct {
		x, y int
	}
	// Count hits per subtile for coloring
	hits := map[[2]int]int{}
	var shards []shardPos

	for elapsed := 0; elapsed < sim.BlizzardDuration; elapsed += sim.ShardSpawnRate {
		dx, dy := sim.ShardOffset(uint32(g.castX), elapsed)
		shards = append(shards, shardPos{g.castX + dx, g.castY + dy})
		for sy := 0; sy < sim.ShardH; sy++ {
			for sx := 0; sx < sim.ShardW; sx++ {
				hits[[2]int{g.castX + dx + sx, g.castY + dy + sy}]++
			}
		}
	}

	// Find max hits for color scaling
	maxHits := 0
	for _, h := range hits {
		if h > maxHits {
			maxHits = h
		}
	}

	// Draw hit subtiles colored by intensity
	for pos, h := range hits {
		if pos[0] < 0 || pos[0] >= sim.FieldW || pos[1] < 0 || pos[1] >= sim.FieldH {
			continue
		}
		t := float64(h) / float64(maxHits)
		r := uint8(80 + t*175)
		b := uint8(180 + t*75)
		clr := color.RGBA{r, 100, b, 200}
		fillDiamond(screen, pos[0], pos[1], 1, 1, clr)
	}

	// Draw shard outlines (2x2 each)
	for _, s := range shards {
		strokeDiamond(screen, s.x, s.y, sim.ShardW, sim.ShardH, color.RGBA{255, 255, 255, 150})
	}

	// Cast point marker
	drawCastMarker(screen, float64(g.castX), float64(g.castY), 8, 2)

	// HUD
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf(
		"Pattern Viewer — touch or hover to move\nCast: (%d, %d)  |  Max overlap: %d  |  25 shards",
		g.castX, g.castY, maxHits), 10, tabBarH+2)
}
