package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

// U-shaped line where GPS on the straight leg can project far along the loop.
func uTurnLine() orb.LineString {
	return orb.LineString{
		{106.634500, -6.134500},
		{106.635500, -6.134000},
		{106.636500, -6.133500},
		{106.636000, -6.134200},
		{106.635500, -6.134800},
		{106.635000, -6.135300},
	}
}

func uTurnStops() []snaptoline.Stop {
	line := uTurnLine()
	return []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[2]},
		{ID: "C", Order: 3, Point: line[5]},
	}
}

func TestSnapContinuityLimitsJumpFromPreviousSnap(t *testing.T) {
	line := uTurnLine()
	stops := uTurnStops()
	cfg := snaptoline.RouteSnapConfig(stops)
	cfg.MaxSnapDistanceMeter = 35
	require.True(t, cfg.SnapContinuityFromPrevious)

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	seed, err := snapper.Snap(snaptoline.GPSPoint{
		Point:   orb.Point{106.635200, -6.134200},
		Bearing: 55,
		Speed:   25,
	})
	require.NoError(t, err)
	seedM := snapper.RouteMeasure(seed.SegmentOrder, seed.Progress)

	// Noisy GPS biased toward the far loop branch.
	r, err := snapper.Snap(snaptoline.GPSPoint{
		Point:   orb.Point{106.635800, -6.134100},
		Bearing: 55,
		Speed:   25,
	})
	require.NoError(t, err)

	nextM := snapper.RouteMeasure(r.SegmentOrder, r.Progress)
	require.Less(t, nextM-seedM, 55.0, "snap should not leap far along route from previous snap")

	jump := snaptoline.DistanceMeter(seed.SnappedPoint, r.SnappedPoint)
	require.Less(t, jump, 55.0, "geodesic snap jump should stay near previous position")
}

func TestSnapContinuityEnabledByDefault(t *testing.T) {
	cfg := snaptoline.RouteSnapConfig(uTurnStops())
	require.True(t, cfg.SnapContinuityFromPrevious)
}

func TestSnapContinuityCreepsForwardWhenGPSMovesButSnapStuck(t *testing.T) {
	line := terminalParallelApproachLine()
	stops := terminalParallelApproachStops()
	cfg := snaptoline.RouteSnapConfig(stops)
	cfg.MaxSnapDistanceMeter = 28
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	_, err = snapper.Snap(snaptoline.GPSPoint{
		Point: orb.Point{106.656310, -6.129220}, Bearing: 137, Speed: 12,
	})
	require.NoError(t, err)
	seed, err := snapper.Snap(snaptoline.GPSPoint{
		Point: orb.Point{106.656260, -6.129180}, Bearing: 137, Speed: 12,
	})
	require.NoError(t, err)
	require.Equal(t, 2, seed.SegmentOrder)
	seedSnap := seed.SnappedPoint

	// GPS drifts toward the parallel branch while moving forward along the route.
	forward := []snaptoline.GPSPoint{
		{Point: orb.Point{106.656268, -6.129172}, Bearing: 137, Speed: 12},
		{Point: orb.Point{106.656276, -6.129164}, Bearing: 137, Speed: 12},
		{Point: orb.Point{106.656284, -6.129156}, Bearing: 137, Speed: 12},
		{Point: orb.Point{106.656292, -6.129148}, Bearing: 137, Speed: 12},
		{Point: orb.Point{106.656300, -6.129140}, Bearing: 137, Speed: 12},
	}
	var moved bool
	var lastSnap orb.Point
	for i, p := range forward {
		r, err := snapper.Snap(p)
		require.NoError(t, err)
		require.Equal(t, 2, r.SegmentOrder)
		if i > 0 {
			jump := snaptoline.DistanceMeter(lastSnap, r.SnappedPoint)
			if jump > 0.5 {
				moved = true
			}
		}
		lastSnap = r.SnappedPoint
	}

	require.True(t, moved, "bus should advance along route when GPS moves but snap was stuck")
	finalJump := snaptoline.DistanceMeter(seedSnap, lastSnap)
	require.Greater(t, finalJump, 0.5, "snap should move from initial stuck position")
	require.Less(t, finalJump, 55.0, "creep should stay within forward cap per sequence")
}
