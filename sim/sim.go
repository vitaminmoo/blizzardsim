package sim

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
)

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

// ShardOffset computes the (dx, dy) offset for a shard spawned at the given
// elapsed frame from a blizzard centered at centerX.
func ShardOffset(centerX uint32, elapsed int) (dx, dy int) {
	var seed D2Seed
	seed.Init(centerX + uint32(elapsed))
	maxRange := int32(BlizzardRadius - 1) // 6
	dx = int(maxRange - seed.RollN(2*maxRange))
	dy = int(maxRange - seed.RollN(2*maxRange))
	return dx, dy
}
