package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestIsSameStop(t *testing.T) {
	a := snaptoline.Stop{ID: "A", Point: orb.Point{106.0, -6.0}}
	b := snaptoline.Stop{ID: "A", Point: orb.Point{106.0, -6.0}}
	require.True(t, snaptoline.IsSameStop(a, b, 10))
}

func TestLoopingRouteSnapperBuild(t *testing.T) {
	line := orb.LineString{
		{106.0, -6.0},
		{106.1, -6.0},
		{106.1, -6.1},
		{106.0, -6.0},
	}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: orb.Point{106.0, -6.0}},
		{ID: "B", Order: 2, Point: orb.Point{106.1, -6.0}},
		{ID: "C", Order: 3, Point: orb.Point{106.1, -6.1}},
		{ID: "A", Order: 4, Point: orb.Point{106.0, -6.0}},
	}

	cfg := snaptoline.Config{
		Looping:                   true,
		AllowSameStartEndStop:     true,
		LoopClosureToleranceMeter: 10,
		MaxSnapDistanceMeter:      60,
		CandidateLimit:            8,
	}

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)
	require.Len(t, snapper.Segments(), 3)
}
