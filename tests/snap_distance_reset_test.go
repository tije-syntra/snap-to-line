package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestSnapDistanceResetWhenDistanceKeepsGrowing(t *testing.T) {
	line := orb.LineString{
		{106.0, -6.0},
		{106.3, -6.0},
	}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}
	cfg := snaptoline.RouteSnapConfig(stops)
	cfg.MaxSnapDistanceMeter = 28
	require.True(t, cfg.SnapDistanceResetOnGrow)

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	_, err = snapper.Snap(snaptoline.GPSPoint{
		Point: orb.Point{106.08, -6.0}, Bearing: 90, Speed: 30,
	})
	require.NoError(t, err)

	drift := []snaptoline.GPSPoint{
		{Point: orb.Point{106.075, -6.005}, Bearing: 90, Speed: 30},
		{Point: orb.Point{106.070, -6.010}, Bearing: 90, Speed: 30},
		{Point: orb.Point{106.065, -6.015}, Bearing: 90, Speed: 30},
		{Point: orb.Point{106.060, -6.020}, Bearing: 90, Speed: 30},
	}

	var reset bool
	var prevDist float64
	for _, p := range drift {
		r, err := snapper.Snap(p)
		require.NoError(t, err)
		if r.HeldReason == "snap_distance_reset" {
			reset = true
		}
		if prevDist > 100 && r.DistanceMeter < prevDist*0.5 {
			reset = true
		}
		prevDist = r.DistanceMeter
	}

	require.True(t, reset, "snap should reset when raw-to-snap distance keeps growing")
	require.Less(t, prevDist, 200.0, "distance should improve after reset")
}

func TestSnapDistanceResetEnabledByDefault(t *testing.T) {
	cfg := snaptoline.RouteSnapConfig(terminalParallelApproachStops())
	require.True(t, cfg.SnapDistanceResetOnGrow)
	require.Equal(t, 2, cfg.SnapDistanceGrowResetTicks)
	require.Equal(t, 100.0, cfg.SnapDistanceResetMaxMeter)
}

func TestSnapDistanceResetImmediateWhenExceedsMaxMeter(t *testing.T) {
	line := orb.LineString{
		{106.0, -6.0},
		{106.3, -6.0},
	}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}
	cfg := snaptoline.RouteSnapConfig(stops,
		snaptoline.WithSnapDistanceResetMaxMeter(100),
		snaptoline.WithSnapDistanceResetOnGrow(false),
	)
	cfg.MaxSnapDistanceMeter = 28

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	_, err = snapper.Snap(snaptoline.GPSPoint{
		Point: orb.Point{106.08, -6.0}, Bearing: 90, Speed: 30,
	})
	require.NoError(t, err)

	// ~444 m north of the last snap — one tick, no grow pattern needed.
	far, err := snapper.Snap(snaptoline.GPSPoint{
		Point: orb.Point{106.08, -6.004}, Bearing: 90, Speed: 30,
	})
	require.NoError(t, err)
	require.Equal(t, "snap_distance_reset", far.HeldReason)
	require.True(t, far.IsOffRoute)
	require.Equal(t, "no candidates within max snap distance", far.RejectedReason)
}
