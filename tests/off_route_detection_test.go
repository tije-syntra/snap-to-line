package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestOffRouteResolvesDefaults(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	require.Equal(t, 200.0, snaptoline.ResolveOffRouteDistanceMeter(cfg))
	require.Equal(t, 5, snaptoline.ResolveOffRouteConsecutiveSamples(cfg))
}

func TestOffRouteUsesExplicitConfig(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.OffRouteDistanceMeter = 100
	cfg.OffRouteConsecutiveSamples = 3
	require.Equal(t, 100.0, snaptoline.ResolveOffRouteDistanceMeter(cfg))
	require.Equal(t, 3, snaptoline.ResolveOffRouteConsecutiveSamples(cfg))
}

func TestUpdateOffRouteCount(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.OffRouteDistanceMeter = 200
	cfg.OffRouteConsecutiveSamples = 5
	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)

	require.False(t, snaptoline.UpdateOffRouteCount(state, 250, cfg))
	require.Equal(t, 1, state.OffRouteCount)

	require.False(t, snaptoline.UpdateOffRouteCount(state, 180, cfg))
	require.Equal(t, 0, state.OffRouteCount)

	for i := 0; i < 5; i++ {
		snaptoline.UpdateOffRouteCount(state, 250, cfg)
	}
	require.Equal(t, 5, state.OffRouteCount)
	require.True(t, snaptoline.UpdateOffRouteCount(state, 250, cfg))
}

func TestApplyConsecutiveOffRouteDetectionDisabled(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.OffRouteDetection = false
	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)

	input := &snaptoline.SnapResult{
		OriginalPoint: orb.Point{106.0, -6.0},
		SnappedPoint:  orb.Point{106.0, -6.0},
		SegmentID:     "seg-1",
		SegmentOrder:  1,
		DistanceMeter: 300,
		Confidence:    0.8,
	}

	out := snaptoline.ApplyConsecutiveOffRouteDetection(state, cfg, input)
	require.Equal(t, input, out)
	require.Equal(t, 0, state.OffRouteCount)
}

func TestSnapperFlagsConsecutiveOffRoute(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.1, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.DefaultConfig()
	cfg.MaxSnapDistanceMeter = 2000
	cfg.OffRouteDetection = true
	cfg.OffRouteDistanceMeter = 200
	cfg.OffRouteConsecutiveSamples = 3
	cfg.HoldLastSegmentOnMiss = false
	cfg.TeleportDetection = false
	cfg.GpsJumpDetection = false

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	farPoint := snaptoline.GPSPoint{Point: orb.Point{106.05, -6.002}}

	first, err := snapper.Snap(farPoint)
	require.NoError(t, err)
	require.False(t, first.IsOffRoute)
	require.Equal(t, 1, first.OffRouteCount)

	second, err := snapper.Snap(farPoint)
	require.NoError(t, err)
	require.False(t, second.IsOffRoute)
	require.Equal(t, 2, second.OffRouteCount)

	third, err := snapper.Snap(farPoint)
	require.NoError(t, err)
	require.True(t, third.IsOffRoute)
	require.Equal(t, 3, third.OffRouteCount)
}

func TestRouteSnapConfigEnablesOffRouteDetection(t *testing.T) {
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: orb.Point{106.0, -6.0}},
		{ID: "B", Order: 2, Point: orb.Point{106.1, -6.0}},
	}
	cfg := snaptoline.RouteSnapConfig(stops)
	require.True(t, cfg.OffRouteDetection)
	require.Equal(t, 200.0, cfg.OffRouteDistanceMeter)
	require.Equal(t, 5, cfg.OffRouteConsecutiveSamples)
}
