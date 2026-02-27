package main

import (
	"flag"
	"fmt"
	"os"

	"blizzardsim/sim"
)

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

	fmt.Printf("Blizzard cast at (%d, %d) — 25 shards (%dx%d each):\n\n", castX, castY, sim.ShardW, sim.ShardH)
	fmt.Printf("  # | Elapsed | Offset     | Position\n")
	fmt.Printf("----+---------+------------+-----------\n")

	shard := 1
	for elapsed := 0; elapsed < sim.BlizzardDuration; elapsed += sim.ShardSpawnRate {
		dx, dy := sim.ShardOffset(uint32(castX), elapsed)
		fmt.Printf(" %2d |     %3d | (%+3d, %+3d) | (%d, %d)\n",
			shard, elapsed, dx, dy, castX+dx, castY+dy)
		shard++
	}
}
