package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
	if sx >= 0 && sx < FieldW && sy >= 0 && sy < FieldH {
		g.valid = true
		g.castX = sx
		g.castY = sy
	}
	// Keep showing last valid pattern when touch ends / cursor leaves grid
	return nil
}

func (g *PatternGame) Draw(screen *ebiten.Image) {
	// Draw isometric grid
	for x := 0; x <= FieldW; x++ {
		x0, y0 := isoToScreen(float64(x), 0)
		x1, y1 := isoToScreen(float64(x), float64(FieldH))
		vector.StrokeLine(screen, x0, y0, x1, y1, 1, colorGrid, false)
	}
	for y := 0; y <= FieldH; y++ {
		x0, y0 := isoToScreen(0, float64(y))
		x1, y1 := isoToScreen(float64(FieldW), float64(y))
		vector.StrokeLine(screen, x0, y0, x1, y1, 1, colorGrid, false)
	}

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

	for elapsed := 0; elapsed < BlizzardDuration; elapsed += ShardSpawnRate {
		var seed D2Seed
		seed.Init(uint32(g.castX) + uint32(elapsed))
		maxRange := int32(BlizzardRadius - 1)
		dx := int(maxRange - seed.RollN(2*maxRange))
		dy := int(maxRange - seed.RollN(2*maxRange))
		shards = append(shards, shardPos{g.castX + dx, g.castY + dy})
		for sy := 0; sy < ShardH; sy++ {
			for sx := 0; sx < ShardW; sx++ {
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
		if pos[0] < 0 || pos[0] >= FieldW || pos[1] < 0 || pos[1] >= FieldH {
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
		strokeDiamond(screen, s.x, s.y, ShardW, ShardH, color.RGBA{255, 255, 255, 150})
	}

	// Cast point marker
	cx, cy := isoToScreen(float64(g.castX)+0.5, float64(g.castY)+0.5)
	d := float32(8)
	vector.StrokeLine(screen, cx-d, cy-d, cx+d, cy+d, 2, colorCastMarker, false)
	vector.StrokeLine(screen, cx-d, cy+d, cx+d, cy-d, 2, colorCastMarker, false)

	// HUD
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf(
		"Pattern Viewer — touch or hover to move\nCast: (%d, %d)  |  Max overlap: %d  |  25 shards",
		g.castX, g.castY, maxHits), 10, tabBarH+2)
}
