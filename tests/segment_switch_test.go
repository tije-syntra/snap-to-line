package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func threeStopLine() orb.LineString {
	return orb.LineString{
		{106.662000, -6.123000},
		{106.662200, -6.122800},
		{106.662400, -6.122600},
	}
}

func threeStopRoute() []snaptoline.Stop {
	line := threeStopLine()
	return []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
		{ID: "C", Order: 3, Point: line[2]},
	}
}

func TestSegmentSwitchBlockedBeforeNextStop(t *testing.T) {
	line := threeStopLine()
	stops := threeStopRoute()
	cfg := snaptoline.RouteSnapConfig(stops)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)
	require.True(t, cfg.RequireNextStopBeforeSegmentSwitch)

	// Lock onto segment 1 between A and B.
	mid, _ := snaptoline.PointAtMeasure(line, 15)
	seed, err := snapper.Snap(snaptoline.GPSPoint{Point: mid, Bearing: 45, Speed: 20})
	require.NoError(t, err)
	require.Equal(t, 1, seed.SegmentOrder)

	// GPS closer to segment 2 but still before stop B (ToMeasure ~31 m on this route).
	beforeB, _ := snaptoline.PointAtMeasure(line, 22)
	r, err := snapper.Snap(snaptoline.GPSPoint{Point: beforeB, Bearing: 45, Speed: 20})
	require.NoError(t, err)
	require.Equal(t, 1, r.SegmentOrder, "segment_id must not change before passing next stop")
	require.Equal(t, seed.SegmentID, r.SegmentID)
}

func TestSegmentSwitchAllowedAfterNextStop(t *testing.T) {
	line := threeStopLine()
	stops := threeStopRoute()
	cfg := snaptoline.RouteSnapConfig(stops)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	mid, _ := snaptoline.PointAtMeasure(line, 15)
	_, err = snapper.Snap(snaptoline.GPSPoint{Point: mid, Bearing: 45, Speed: 20})
	require.NoError(t, err)

	// Past stop B.
	pastB, _ := snaptoline.PointAtMeasure(line, 38)
	r, err := snapper.Snap(snaptoline.GPSPoint{Point: pastB, Bearing: 45, Speed: 20})
	require.NoError(t, err)
	require.Equal(t, 2, r.SegmentOrder)
	require.NotEqual(t, "", r.SegmentID)
}

func TestRouteSnapConfigEnablesNextStopSegmentGuard(t *testing.T) {
	cfg := snaptoline.RouteSnapConfig(threeStopRoute())
	require.True(t, cfg.RequireNextStopBeforeSegmentSwitch)
	require.Greater(t, cfg.NextStopPassToleranceMeter, 0.0)

	disabled := snaptoline.RouteSnapConfig(threeStopRoute(), snaptoline.WithRequireNextStopBeforeSegmentSwitch(false))
	require.False(t, disabled.RequireNextStopBeforeSegmentSwitch)
}
