package main

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	hmDefaultSamples = 100_000
	// Isometric tile size for the heatmap grid — large enough for text labels
	hmTileW = 40
	hmTileH = 20
)

// Shard offsets range from -6 to +5, covering subtiles -6 to +6 (13 wide).
const gridSize = 13

func gridIdx(dx, dy int) int {
	return (dy+6)*gridSize + (dx + 6)
}

type gridArray = [gridSize * gridSize]uint32

// Control presets
var (
	hmStartPresets  = []uint64{0, 100, 1000, 10000, 50000}
	hmEndPresets    = []uint64{999, 9999, 99999, 999999, 9999999}
	hmStepPresets   = []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 25}
	hmSpeedValues   = []time.Duration{0, time.Millisecond, 10 * time.Millisecond, 100 * time.Millisecond, 500 * time.Millisecond}
	hmSpeedLabels   = []string{"Max", "1ms", "10ms", "100ms", "500ms"}
	hmWindowPresets = []int{0, 1, 5, 10, 50, 100, 500, 1000}
)

// Control button layout
const (
	hmControlH = 24
	hmBtnGap   = 8
)

// hmControlY returns the Y position of the control row (below the grid).
func hmControlY() int {
	// Grid bottom is at approximately (gridSize-1+gridSize-1)*hmTileH + hmGridTopY
	return hmGridTopY() + (gridSize+gridSize-1)*hmTileH + 20
}

func hmGridTopY() int {
	return 20 + tabBarH
}

// heatColor maps a normalized value [0,1] to a black→red gradient.
func heatColor(t float64) color.RGBA {
	if t <= 0 {
		return color.RGBA{0, 0, 0, 255}
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{uint8(t * 255), 0, 0, 255}
}

// simulateBlizzardGrid returns per-subtile hit counts for a single X value.
func simulateBlizzardGrid(centerX uint32) gridArray {
	var grid gridArray
	for elapsed := 0; elapsed < BlizzardDuration; elapsed += ShardSpawnRate {
		var seed D2Seed
		seed.Init(centerX + uint32(elapsed))
		maxRange := int32(BlizzardRadius - 1)
		dx := int(maxRange - seed.RollN(2*maxRange))
		dy := int(maxRange - seed.RollN(2*maxRange))
		for sy := 0; sy < ShardH; sy++ {
			for sx := 0; sx < ShardW; sx++ {
				gx := dx + sx
				gy := dy + sy
				if gx >= -6 && gx <= 6 && gy >= -6 && gy <= 6 {
					grid[gridIdx(gx, gy)]++
				}
			}
		}
	}
	return grid
}

type HeatmapGame struct {
	grid    [gridSize * gridSize]uint64 // accumulated hits per subtile
	maxVal  uint64
	minVal  uint64
	samples uint64

	// Sliding window (nil = no window, accumulate all)
	history []gridArray // ring buffer of past cast grids
	histIdx int         // next write position in ring buffer
	histLen int         // current number of entries in ring buffer
	winSize int         // window capacity (0 = unlimited)

	// Animation state
	startX   uint64
	endX     uint64
	nextX    uint64        // next X to simulate
	step     uint64        // increment per cast (default 1)
	interval time.Duration // time between adding casts (0 = max speed)
	lastAdd  time.Time
	done     bool
	paused   bool

	// Control preset indices
	startIdx  int
	endIdx    int
	stepIdx   int
	speedIdx  int
	windowIdx int
}

func NewHeatmapGame() *HeatmapGame {
	g := &HeatmapGame{
		endIdx:    2, // 99999
		speedIdx:  3, // 100ms
		windowIdx: 3, // 10
	}
	g.Reset()
	return g
}

func (g *HeatmapGame) Reset() {
	g.startX = hmStartPresets[g.startIdx]
	g.endX = hmEndPresets[g.endIdx]
	g.step = hmStepPresets[g.stepIdx]
	g.interval = hmSpeedValues[g.speedIdx]
	g.winSize = hmWindowPresets[g.windowIdx]

	g.grid = [gridSize * gridSize]uint64{}
	g.maxVal = 0
	g.minVal = 0
	g.samples = 0
	g.nextX = g.startX
	g.done = false
	g.paused = false
	g.lastAdd = time.Now()

	if g.winSize > 0 {
		g.history = make([]gridArray, g.winSize)
	} else {
		g.history = nil
	}
	g.histIdx = 0
	g.histLen = 0
}

func (g *HeatmapGame) recalcMinMax() {
	g.minVal = math.MaxUint64
	g.maxVal = 0
	for _, v := range g.grid {
		if v < g.minVal {
			g.minVal = v
		}
		if v > g.maxVal {
			g.maxVal = v
		}
	}
}

func (g *HeatmapGame) addCast(x uint32) {
	cast := simulateBlizzardGrid(x)

	// If windowed, subtract the oldest entry when the buffer is full
	if g.winSize > 0 {
		if g.histLen == g.winSize {
			old := g.history[g.histIdx]
			for i, v := range old {
				g.grid[i] -= uint64(v)
			}
			g.samples--
		}
		g.history[g.histIdx] = cast
		g.histIdx = (g.histIdx + 1) % g.winSize
		if g.histLen < g.winSize {
			g.histLen++
		}
	}

	for i, v := range cast {
		g.grid[i] += uint64(v)
	}
	g.samples++
}

func (g *HeatmapGame) updateControls() {
	ctrlY := hmControlY()
	x := 10
	changed := false

	// Start X
	w := 110
	if btnClicked(x, ctrlY, w, hmControlH) {
		cycleIdx(&g.startIdx, len(hmStartPresets), true)
		changed = true
	} else if btnRightClicked(x, ctrlY, w, hmControlH) {
		cycleIdx(&g.startIdx, len(hmStartPresets), false)
		changed = true
	}
	x += w + hmBtnGap

	// End X
	w = 130
	if btnClicked(x, ctrlY, w, hmControlH) {
		cycleIdx(&g.endIdx, len(hmEndPresets), true)
		changed = true
	} else if btnRightClicked(x, ctrlY, w, hmControlH) {
		cycleIdx(&g.endIdx, len(hmEndPresets), false)
		changed = true
	}
	x += w + hmBtnGap

	// Step
	w = 90
	if btnClicked(x, ctrlY, w, hmControlH) {
		cycleIdx(&g.stepIdx, len(hmStepPresets), true)
		changed = true
	} else if btnRightClicked(x, ctrlY, w, hmControlH) {
		cycleIdx(&g.stepIdx, len(hmStepPresets), false)
		changed = true
	}
	x += w + hmBtnGap

	// Speed
	w = 110
	if btnClicked(x, ctrlY, w, hmControlH) {
		cycleIdx(&g.speedIdx, len(hmSpeedValues), true)
		changed = true
	} else if btnRightClicked(x, ctrlY, w, hmControlH) {
		cycleIdx(&g.speedIdx, len(hmSpeedValues), false)
		changed = true
	}
	x += w + hmBtnGap

	// Window
	w = 100
	if btnClicked(x, ctrlY, w, hmControlH) {
		cycleIdx(&g.windowIdx, len(hmWindowPresets), true)
		changed = true
	} else if btnRightClicked(x, ctrlY, w, hmControlH) {
		cycleIdx(&g.windowIdx, len(hmWindowPresets), false)
		changed = true
	}
	x += w + hmBtnGap

	// Pause/Play
	w = 70
	if btnClicked(x, ctrlY, w, hmControlH) {
		g.paused = !g.paused
		if !g.paused {
			g.lastAdd = time.Now()
		}
	}
	x += w + hmBtnGap

	// Reset
	w = 70
	if btnClicked(x, ctrlY, w, hmControlH) {
		g.Reset()
	}

	if changed {
		g.Reset()
	}
}

func (g *HeatmapGame) drawControls(screen *ebiten.Image) {
	ctrlY := hmControlY()
	x := 10

	// Start X
	w := 110
	drawButton(screen, x, ctrlY, w, hmControlH,
		fmt.Sprintf("Start:%d", hmStartPresets[g.startIdx]), false)
	x += w + hmBtnGap

	// End X
	w = 130
	drawButton(screen, x, ctrlY, w, hmControlH,
		fmt.Sprintf("End:%d", hmEndPresets[g.endIdx]), false)
	x += w + hmBtnGap

	// Step
	w = 90
	drawButton(screen, x, ctrlY, w, hmControlH,
		fmt.Sprintf("Step:%d", hmStepPresets[g.stepIdx]), false)
	x += w + hmBtnGap

	// Speed
	w = 110
	drawButton(screen, x, ctrlY, w, hmControlH,
		fmt.Sprintf("Speed:%s", hmSpeedLabels[g.speedIdx]), false)
	x += w + hmBtnGap

	// Window
	w = 100
	winLabel := "Win:off"
	if hmWindowPresets[g.windowIdx] > 0 {
		winLabel = fmt.Sprintf("Win:%d", hmWindowPresets[g.windowIdx])
	}
	drawButton(screen, x, ctrlY, w, hmControlH, winLabel, false)
	x += w + hmBtnGap

	// Pause/Play
	w = 70
	pauseLabel := "Pause"
	if g.paused {
		pauseLabel = "Play"
	}
	drawButton(screen, x, ctrlY, w, hmControlH, pauseLabel, g.paused)
	x += w + hmBtnGap

	// Reset
	w = 70
	drawButton(screen, x, ctrlY, w, hmControlH, "Reset", false)
}

func (g *HeatmapGame) Update() error {
	g.updateControls()

	if g.done || g.paused {
		return nil
	}

	if g.interval == 0 {
		// Max speed: process a large batch per frame
		for i := 0; i < 10000 && !g.done; i++ {
			g.addCast(uint32(g.nextX))
			g.nextX += g.step
			if g.nextX > g.endX {
				g.done = true
			}
		}
		g.recalcMinMax()
		return nil
	}

	// Time-based animation
	now := time.Now()
	elapsed := now.Sub(g.lastAdd)
	n := int(elapsed / g.interval)
	if n <= 0 {
		return nil
	}
	g.lastAdd = g.lastAdd.Add(time.Duration(n) * g.interval)

	for i := 0; i < n && !g.done; i++ {
		g.addCast(uint32(g.nextX))
		g.nextX += g.step
		if g.nextX > g.endX {
			g.done = true
		}
	}
	g.recalcMinMax()
	return nil
}

func (g *HeatmapGame) hmIsoToScreen(gx, gy float64) (float32, float32) {
	centerX := float64(gridSize+gridSize)*float64(hmTileW)/2 + 10
	centerY := float64(hmGridTopY())
	sx := (gx-gy)*float64(hmTileW) + centerX
	sy := (gx+gy)*float64(hmTileH) + centerY
	return float32(sx), float32(sy)
}

func isEdge(dx, dy int) bool {
	return dx == -6 || dx == 6 || dy == -6 || dy == 6
}

func (g *HeatmapGame) Draw(screen *ebiten.Image) {
	// Compute color min/max excluding edge/corner cells
	colorMin := uint64(math.MaxUint64)
	colorMax := uint64(0)
	for dy := -6; dy <= 6; dy++ {
		for dx := -6; dx <= 6; dx++ {
			if isEdge(dx, dy) {
				continue
			}
			v := g.grid[gridIdx(dx, dy)]
			if v < colorMin {
				colorMin = v
			}
			if v > colorMax {
				colorMax = v
			}
		}
	}
	valRange := float64(colorMax - colorMin)
	if valRange == 0 {
		valRange = 1
	}

	for dy := -6; dy <= 6; dy++ {
		for dx := -6; dx <= 6; dx++ {
			idx := gridIdx(dx, dy)
			v := g.grid[idx]

			var clr color.RGBA
			if isEdge(dx, dy) {
				clr = color.RGBA{50, 50, 50, 255}
			} else {
				t := float64(v-colorMin) / valRange
				clr = heatColor(t)
			}

			gx := dx + 6
			gy := dy + 6

			topX, topY := g.hmIsoToScreen(float64(gx), float64(gy))
			rightX, rightY := g.hmIsoToScreen(float64(gx+1), float64(gy))
			bottomX, bottomY := g.hmIsoToScreen(float64(gx+1), float64(gy+1))
			leftX, leftY := g.hmIsoToScreen(float64(gx), float64(gy+1))

			var path vector.Path
			path.MoveTo(topX, topY)
			path.LineTo(rightX, rightY)
			path.LineTo(bottomX, bottomY)
			path.LineTo(leftX, leftY)
			path.Close()

			cs := colorScaleFrom(clr)
			vector.FillPath(screen, &path, nil, &vector.DrawPathOptions{ColorScale: cs})
			vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: 0.5}, &vector.DrawPathOptions{
				ColorScale: colorScaleFrom(color.RGBA{80, 80, 80, 255}),
			})

			var label string
			if g.samples > 0 {
				pct := float64(v) / float64(g.samples) * 100
				label = fmt.Sprintf("%.1f%%", pct)
			} else {
				label = "0%"
			}
			cx := (topX + bottomX) / 2
			cy := (topY + bottomY) / 2
			charW := float32(6)
			charH := float32(16)
			lx := int(cx - charW*float32(len(label))/2)
			ly := int(cy - charH/2)
			ebitenutil.DebugPrintAt(screen, label, lx, ly)
		}
	}

	// Status bar
	statusY := hmControlY() + hmControlH + 8
	status := ""
	if !g.done && !g.paused {
		status = fmt.Sprintf("  |  X=%d", g.nextX)
	} else if g.paused {
		status = "  |  PAUSED"
	} else {
		status = "  |  DONE"
	}
	winInfo := ""
	if g.winSize > 0 {
		winInfo = fmt.Sprintf("  |  Window: %d", g.winSize)
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf(
		"Heatmap: %d casts  |  Min: %d  Max: %d%s%s  (L/R click buttons to change)",
		g.samples, g.minVal, g.maxVal, winInfo, status), 10, statusY)

	// Controls
	g.drawControls(screen)
}
