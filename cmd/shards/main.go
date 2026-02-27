package main

import (
	"flag"
	"fmt"
	"os"
)

// D2 game constants (must match main package)
const (
	BlizzardDuration = 100
	ShardSpawnRate   = 4
	ShardW           = 2
	ShardH           = 2
	BlizzardRadius   = 7
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

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: shards <x> <y>\n\nComputes the 25 shard (subprojectile) positions for a Blizzard cast at the given subtile coordinates.\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	var castX, castY int
	if _, err := fmt.Sscan(flag.Arg(0), &castX); err != nil {
		fmt.Fprintf(os.Stderr, "invalid x coordinate: %s\n", flag.Arg(0))
		os.Exit(1)
	}
	if _, err := fmt.Sscan(flag.Arg(1), &castY); err != nil {
		fmt.Fprintf(os.Stderr, "invalid y coordinate: %s\n", flag.Arg(1))
		os.Exit(1)
	}

	fmt.Printf("Blizzard cast at (%d, %d) — 25 shards (%dx%d each):\n\n", castX, castY, ShardW, ShardH)
	fmt.Printf("  # | Elapsed | Offset     | Position\n")
	fmt.Printf("----+---------+------------+-----------\n")

	shard := 1
	for elapsed := 0; elapsed < BlizzardDuration; elapsed += ShardSpawnRate {
		var seed D2Seed
		seed.Init(uint32(castX) + uint32(elapsed))
		maxRange := int32(BlizzardRadius - 1) // 6
		dx := int(maxRange - seed.RollN(2*maxRange))
		dy := int(maxRange - seed.RollN(2*maxRange))
		fmt.Printf(" %2d |     %3d | (%+3d, %+3d) | (%d, %d)\n",
			shard, elapsed, dx, dy, castX+dx, castY+dy)
		shard++
	}
}
