package main

import (
	"fmt"
	"log"

	"github.com/paulmach/orb"
	snaptoline "github.com/tije-syntra/snap-to-line"
)

func main() {
	line := orb.LineString{
		{106.0, -6.0},
		{106.05, -6.0},
		{106.1, -6.0},
		{106.1, -6.05},
		{106.0, -6.0},
	}

	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: orb.Point{106.0, -6.0}},
		{ID: "B", Order: 2, Point: orb.Point{106.1, -6.0}},
		{ID: "C", Order: 3, Point: orb.Point{106.1, -6.05}},
		{ID: "A", Order: 4, Point: orb.Point{106.0, -6.0}},
	}

	cfg := snaptoline.Config{
		Looping:                   true,
		AllowSameStartEndStop:     true,
		LoopClosureToleranceMeter: 10,
		MaxSnapDistanceMeter:      60,
		CandidateLimit:            8,
		UseBearingValidation:      true,
		MaxBearingDiffDegree:      60,
	}

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	if err != nil {
		log.Fatal(err)
	}

	points := []snaptoline.GPSPoint{
		{Point: orb.Point{106.02, -6.0}, Bearing: 90},
		{Point: orb.Point{106.08, -6.0}, Bearing: 90},
		{Point: orb.Point{106.1, -6.02}, Bearing: 180},
	}

	for i, point := range points {
		result, err := snapper.Snap(point)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(
			"point %d: segment=%s order=%d dist=%.1fm confidence=%.2f offRoute=%v\n",
			i+1,
			result.SegmentID,
			result.SegmentOrder,
			result.DistanceMeter,
			result.Confidence,
			result.IsOffRoute,
		)
	}
}
