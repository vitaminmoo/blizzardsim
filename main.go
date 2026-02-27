package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// D2 game constants
const (
	TPS              = 25  // D2 game tick rate
	FieldW           = 50  // subtiles wide
	FieldH           = 35  // subtiles tall
	BlizzardDuration = 100 // frames
	ShardSpawnRate   = 4   // spawn every N frames
	ShardLifetime    = 9   // frames
	ShardW           = 2   // subtiles
	ShardH           = 2   // subtiles
	BlizzardRadius   = 7
	BlizzardDelay    = 45 // frames — Blizzard's NextDelay (1.8s)
	ActionFrame105   = 5  // cast animation frames at 105 FCR (63-199 FCR range)
	BlizzardCastRate = BlizzardDelay + ActionFrame105 // 50 frames between casts

	// Isometric projection constants (2:1 ratio)
	HalfTileW = 14
	HalfTileH = 7
	ScreenW   = (FieldW + FieldH) * HalfTileW + 20        // 1210
	ScreenH   = (FieldW+FieldH)*HalfTileH + 50 + tabBarH  // 675
	OffsetX   = FieldH*HalfTileW + 10                      // 500
	OffsetY   = 25 + tabBarH                               // 55
)

// Colors
var (
	colorGrid       = color.RGBA{40, 40, 40, 255}
	colorMonsterPt  = color.RGBA{0, 220, 0, 255}   // POINT (size 1)
	colorMonsterSm  = color.RGBA{220, 220, 0, 255} // SMALL (size 2)
	colorMonsterBig = color.RGBA{255, 140, 0, 255} // BIG   (size 3)
	colorShardNorm  = color.RGBA{100, 200, 255, 255}
	colorShardHit   = color.RGBA{255, 50, 180, 255}
	colorShardEdge  = color.RGBA{255, 255, 255, 200}
	colorCastMarker = color.RGBA{200, 220, 255, 255}
)

// isoToScreen converts game coordinates to screen pixel coordinates.
func isoToScreen(gx, gy float64) (float32, float32) {
	sx := (gx-gy)*float64(HalfTileW) + float64(OffsetX)
	sy := (gx+gy)*float64(HalfTileH) + float64(OffsetY)
	return float32(sx), float32(sy)
}

// screenToIso converts screen pixel coordinates to game coordinates.
func screenToIso(sx, sy int) (float64, float64) {
	dx := float64(sx) - float64(OffsetX)
	dy := float64(sy) - float64(OffsetY)
	gx := (dy/float64(HalfTileH) + dx/float64(HalfTileW)) / 2
	gy := (dy/float64(HalfTileH) - dx/float64(HalfTileW)) / 2
	return gx, gy
}

// diamondPath builds a vector.Path for an isometric diamond at game coords (gx,gy) with size (gw,gh).
func diamondPath(gx, gy, gw, gh int) vector.Path {
	// Four corners of the diamond: top, right, bottom, left
	topX, topY := isoToScreen(float64(gx), float64(gy))
	rightX, rightY := isoToScreen(float64(gx+gw), float64(gy))
	bottomX, bottomY := isoToScreen(float64(gx+gw), float64(gy+gh))
	leftX, leftY := isoToScreen(float64(gx), float64(gy+gh))

	var path vector.Path
	path.MoveTo(topX, topY)
	path.LineTo(rightX, rightY)
	path.LineTo(bottomX, bottomY)
	path.LineTo(leftX, leftY)
	path.Close()
	return path
}

// colorScaleFrom converts a color.Color to an ebiten.ColorScale.
func colorScaleFrom(clr color.Color) ebiten.ColorScale {
	r, g, b, a := clr.RGBA()
	var cs ebiten.ColorScale
	if a == 0 {
		return cs
	}
	fa := float32(a) / 0xffff
	cs.SetR(float32(r) / 0xffff / fa)
	cs.SetG(float32(g) / 0xffff / fa)
	cs.SetB(float32(b) / 0xffff / fa)
	cs.SetA(fa)
	return cs
}

// fillDiamond draws a filled isometric diamond.
func fillDiamond(screen *ebiten.Image, gx, gy, gw, gh int, clr color.Color) {
	path := diamondPath(gx, gy, gw, gh)
	vector.FillPath(screen, &path, nil, &vector.DrawPathOptions{
		ColorScale: colorScaleFrom(clr),
	})
}

// strokeDiamond draws an isometric diamond outline.
func strokeDiamond(screen *ebiten.Image, gx, gy, gw, gh int, clr color.Color) {
	path := diamondPath(gx, gy, gw, gh)
	vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: 1}, &vector.DrawPathOptions{
		ColorScale: colorScaleFrom(clr),
	})
}

// D2Seed — exact D2 LCG from D2MOO
type D2Seed struct {
	Lo uint32
	Hi uint32
}

func (s *D2Seed) Init(low uint32) {
	s.Lo = low
	s.Hi = 666
}

func (s *D2Seed) Roll() uint32 {
	result := uint64(s.Hi) + uint64(0x6AC690C5)*uint64(s.Lo)
	s.Lo = uint32(result)
	s.Hi = uint32(result >> 32)
	return s.Lo
}

func (s *D2Seed) RollN(n int32) int32 {
	if n <= 0 {
		return 0
	}
	r := s.Roll()
	if n&(n-1) == 0 {
		return int32(r & uint32(n-1))
	}
	return int32(r % uint32(n))
}

// Shard is a single blizzard sub-projectile
type Shard struct {
	X, Y       int // top-left subtile coords
	FramesLeft int
	Hit        bool
}

// Blizzard is a cast instance
type Blizzard struct {
	CenterX, CenterY int // subtile coords
	FramesLeft        int
	Shards            []Shard
}

// D2 collision unit sizes (from D2Collision.h COLLISION_UNIT_SIZE enum)
const (
	CollisionSizePoint = 1 // 1 subtile
	CollisionSizeSmall = 2 // plus/cross (5 cells)
	CollisionSizeBig   = 3 // 3x3 square (9 cells)
)

// MonsterHitbox represents a monster on the field
type MonsterHitbox struct {
	X, Y float64 // center position in subtiles
	Size int     // CollisionSizePoint/Small/Big (1/2/3)
	VelX float64
	MinX float64
	MaxX float64
}

// Cells returns the occupied subtile rectangles (x, y, w, h) for the monster.
// Matches D2's COLLISION_CheckMaskWithSize logic.
func (m *MonsterHitbox) Cells() [][4]int {
	cx := int(math.Floor(m.X))
	cy := int(math.Floor(m.Y))
	switch m.Size {
	case CollisionSizePoint:
		return [][4]int{{cx, cy, 1, 1}}
	case CollisionSizeSmall:
		// Plus/cross shape: center + 4 cardinal neighbors
		return [][4]int{
			{cx, cy - 1, 1, 1},
			{cx - 1, cy, 1, 1},
			{cx, cy, 1, 1},
			{cx + 1, cy, 1, 1},
			{cx, cy + 1, 1, 1},
		}
	case CollisionSizeBig:
		// 3x3 square grid
		cells := make([][4]int, 0, 9)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				cells = append(cells, [4]int{cx + dx, cy + dy, 1, 1})
			}
		}
		return cells
	default:
		return [][4]int{{cx, cy, 1, 1}}
	}
}

// Game implements the live blizzard simulator tab.
type Game struct {
	Blizzards    []*Blizzard
	Monsters     [6]MonsterHitbox
	CastingLeft  int // action frame countdown (blizzard spawns when this hits 0)
	CooldownLeft int // NextDelay countdown after cast completes
	PendingX     int // queued blizzard center
	PendingY     int
}

func NewGame() *Game {
	g := &Game{}
	// POINT (size 1): stationary left, patrol right — no overlap
	g.Monsters[0] = MonsterHitbox{
		X:    7,
		Y:    8,
		Size: CollisionSizePoint,
	}
	g.Monsters[1] = MonsterHitbox{
		X:    28,
		Y:    8,
		Size: CollisionSizePoint,
		VelX: 0.08,
		MinX: 15,
		MaxX: 42,
	}
	// SMALL (size 2, plus shape): stationary left, patrol right
	g.Monsters[2] = MonsterHitbox{
		X:    7,
		Y:    17,
		Size: CollisionSizeSmall,
	}
	g.Monsters[3] = MonsterHitbox{
		X:    28,
		Y:    17,
		Size: CollisionSizeSmall,
		VelX: 0.08,
		MinX: 15,
		MaxX: 42,
	}
	// BIG (size 3, 3x3): stationary left, patrol right
	g.Monsters[4] = MonsterHitbox{
		X:    7,
		Y:    28,
		Size: CollisionSizeBig,
	}
	g.Monsters[5] = MonsterHitbox{
		X:    28,
		Y:    28,
		Size: CollisionSizeBig,
		VelX: 0.08,
		MinX: 15,
		MaxX: 42,
	}
	return g
}

func (g *Game) Update() error {
	// Casting phase: action frame countdown, spawn blizzard when it finishes
	if g.CastingLeft > 0 {
		g.CastingLeft--
		if g.CastingLeft == 0 {
			g.Blizzards = append(g.Blizzards, &Blizzard{
				CenterX:    g.PendingX,
				CenterY:    g.PendingY,
				FramesLeft: BlizzardDuration,
			})
			g.CooldownLeft = BlizzardDelay
		}
	}

	// Cooldown tick
	if g.CooldownLeft > 0 {
		g.CooldownLeft--
	}

	// Mouse click or touch: hold to spam, gated by casting + cooldown
	mx, my := -1, -1
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		mx, my = ebiten.CursorPosition()
	} else if tids := ebiten.AppendTouchIDs(nil); len(tids) > 0 {
		mx, my = ebiten.TouchPosition(tids[0])
	}
	if mx >= 0 && g.CastingLeft == 0 && g.CooldownLeft == 0 {
		gx, gy := screenToIso(mx, my)
		sx := int(math.Floor(gx))
		sy := int(math.Floor(gy))
		// Only cast if within field bounds (ignores clicks on tab bar)
		if sx >= 0 && sy >= 0 && sx < FieldW && sy < FieldH {
			g.PendingX = sx
			g.PendingY = sy
			g.CastingLeft = ActionFrame105
		}
	}

	// Monster movement — all monsters with velocity
	for i := range g.Monsters {
		m := &g.Monsters[i]
		if m.VelX != 0 {
			m.X += m.VelX
			if m.X >= m.MaxX {
				m.X = m.MaxX
				m.VelX = -m.VelX
			} else if m.X <= m.MinX {
				m.X = m.MinX
				m.VelX = -m.VelX
			}
		}
	}

	// Blizzard ticks
	alive := g.Blizzards[:0]
	for _, b := range g.Blizzards {
		// Spawn shard
		if b.FramesLeft > 0 && b.FramesLeft%ShardSpawnRate == 0 {
			var seed D2Seed
			seed.Init(uint32(b.CenterX) + uint32(b.FramesLeft))
			maxRange := int32(BlizzardRadius - 1) // 6
			offsetX := seed.RollN(2*maxRange) - maxRange
			offsetY := seed.RollN(2*maxRange) - maxRange
			b.Shards = append(b.Shards, Shard{
				X:          b.CenterX + int(offsetX),
				Y:          b.CenterY + int(offsetY),
				FramesLeft: ShardLifetime,
			})
		}
		if b.FramesLeft > 0 {
			b.FramesLeft--
		}

		// Update shards
		shardAlive := b.Shards[:0]
		for i := range b.Shards {
			s := &b.Shards[i]
			// Hit detection against all monsters
			if !s.Hit {
				for mi := range g.Monsters {
					for _, cell := range g.Monsters[mi].Cells() {
						if rectsOverlap(s.X, s.Y, ShardW, ShardH, cell[0], cell[1], cell[2], cell[3]) {
							s.Hit = true
							break
						}
					}
					if s.Hit {
						break
					}
				}
			}
			s.FramesLeft--
			if s.FramesLeft > 0 {
				shardAlive = append(shardAlive, *s)
			}
		}
		b.Shards = shardAlive

		if b.FramesLeft > 0 || len(b.Shards) > 0 {
			alive = append(alive, b)
		}
	}
	g.Blizzards = alive

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Isometric grid lines
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

	// Monsters
	for i := range g.Monsters {
		m := &g.Monsters[i]
		var c color.Color
		switch m.Size {
		case CollisionSizePoint:
			c = colorMonsterPt
		case CollisionSizeSmall:
			c = colorMonsterSm
		case CollisionSizeBig:
			c = colorMonsterBig
		default:
			c = colorMonsterPt
		}
		for _, cell := range m.Cells() {
			strokeDiamond(screen, cell[0], cell[1], cell[2], cell[3], c)
		}
	}

	// Blizzards
	for _, b := range g.Blizzards {
		// Cast point X marker
		if b.FramesLeft > 0 {
			cx, cy := isoToScreen(float64(b.CenterX)+0.5, float64(b.CenterY)+0.5)
			d := float32(6)
			vector.StrokeLine(screen, cx-d, cy-d, cx+d, cy+d, 1, colorCastMarker, false)
			vector.StrokeLine(screen, cx-d, cy+d, cx+d, cy-d, 1, colorCastMarker, false)
		}

		// Shards
		for _, s := range b.Shards {
			c := colorShardNorm
			if s.Hit {
				c = colorShardHit
			}
			fillDiamond(screen, s.X, s.Y, ShardW, ShardH, c)
			strokeDiamond(screen, s.X, s.Y, ShardW, ShardH, colorShardEdge)
		}
	}

	// HUD
	activeShards := 0
	for _, b := range g.Blizzards {
		activeShards += len(b.Shards)
	}
	status := "READY"
	if g.CastingLeft > 0 {
		status = fmt.Sprintf("CASTING (%d)", g.CastingLeft)
	} else if g.CooldownLeft > 0 {
		status = fmt.Sprintf("COOLDOWN (%d)", g.CooldownLeft)
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf(
		"Click/tap to cast (105 FCR)\nBlizzards: %d\nShards:    %d\nStatus:    %s",
		len(g.Blizzards), activeShards, status), 10, tabBarH+2)
}

func rectsOverlap(ax, ay, aw, ah, bx, by, bw, bh int) bool {
	return ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by
}

func main() {
	ebiten.SetWindowSize(ScreenW, ScreenH)
	ebiten.SetWindowTitle("Blizzard Simulator")
	ebiten.SetTPS(TPS)

	if err := ebiten.RunGame(NewApp()); err != nil {
		log.Fatal(err)
	}
}
