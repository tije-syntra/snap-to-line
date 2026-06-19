package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

// Route A→B→A2 with return geometry overlapping near terminal B.
func terminalOverlapLine() orb.LineString {
	return orb.LineString{
		{106.654700, -6.129700},
		{106.654760, -6.130080},
		{106.654820, -6.130450},
		{106.654760, -6.130080},
		{106.654700, -6.129700},
	}
}

func terminalOverlapStops() []snaptoline.Stop {
	line := terminalOverlapLine()
	return []snaptoline.Stop{
		{ID: "A", Name: "Terminal A", Order: 1, Point: line[0]},
		{ID: "B", Name: "Terminal B", Order: 2, Point: line[1]},
		{ID: "A2", Name: "Terminal A return", Order: 3, Point: line[2]},
	}
}

func TestPreventBackwardTransitionAtTerminalOverlap(t *testing.T) {
	line := terminalOverlapLine()
	stops := terminalOverlapStops()

	cfg := snaptoline.RouteSnapConfig(stops)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	// Advance along A→B→A2 to segment 2.
	advance := []snaptoline.GPSPoint{
		{Point: orb.Point{106.654730, -6.129850}, Bearing: 180, Speed: 20},
		{Point: orb.Point{106.654760, -6.130050}, Bearing: 180, Speed: 20},
		{Point: orb.Point{106.654800, -6.130350}, Bearing: 180, Speed: 15},
	}
	var lastOrder int
	for _, p := range advance {
		r, err := snapper.Snap(p)
		require.NoError(t, err)
		lastOrder = r.SegmentOrder
	}
	require.Equal(t, 2, lastOrder)

	// Noisy GPS near terminal B that would snap to reverse geometry on segment 1.
	p2 := snaptoline.GPSPoint{Point: orb.Point{106.654714, -6.129853}, Bearing: 0, Speed: 5}
	r2, err := snapper.Snap(p2)
	require.NoError(t, err)
	require.GreaterOrEqual(t, r2.SegmentOrder, lastOrder,
		"backward segment jump should be rejected at terminal overlap")
}

func TestRouteSnapConfigEnablesBackwardGuard(t *testing.T) {
	stops := terminalOverlapStops()
	cfg := snaptoline.RouteSnapConfig(stops)
	require.True(t, cfg.PreventBackwardTransition)
	require.Greater(t, cfg.MeasureRegressionToleranceMeter, 0.0)
	require.Greater(t, cfg.ClampBackwardMinConfidence, 0.0)
	require.Greater(t, cfg.ClampDwellSpeedKmh, 0.0)
}
