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
	require.True(t, cfg.RequireStopRadiusForSegmentSwitch)

	mid, _ := snaptoline.PointAtMeasure(line, 15)
	_, err = snapper.Snap(snaptoline.GPSPoint{Point: mid, Bearing: 45, Speed: 20})
	require.NoError(t, err)

	// Past stop B on segment 2, still within halte radius (~6 m from gate B).
	pastB, _ := snaptoline.PointAtMeasure(line, 38)
	gate := stops[1].Point
	require.Less(t, snaptoline.DistanceMeter(pastB, gate), 20.0)
	r, err := snapper.Snap(snaptoline.GPSPoint{Point: pastB, Bearing: 45, Speed: 20})
	require.NoError(t, err)
	require.Equal(t, 2, r.SegmentOrder)
	require.NotEqual(t, "", r.SegmentID)
}

func TestSegmentSwitchBlockedOutsideStopRadius(t *testing.T) {
	line := threeStopLine()
	stops := threeStopRoute()
	cfg := snaptoline.RouteSnapConfig(stops, snaptoline.WithSegmentSwitchStopRadiusMeter(20))
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	mid, _ := snaptoline.PointAtMeasure(line, 15)
	seed, err := snapper.Snap(snaptoline.GPSPoint{Point: mid, Bearing: 45, Speed: 20})
	require.NoError(t, err)
	require.Equal(t, 1, seed.SegmentOrder)

	// On route past B by measure, but ~55 m from stop B — no segment switch.
	// Bus never entered halte radius, so departure latch does not apply.
	pastB, _ := snaptoline.PointAtMeasure(line, 40)
	farFromB := orb.Point{pastB[0] + 0.0005, pastB[1]}
	r, err := snapper.Snap(snaptoline.GPSPoint{Point: farFromB, Bearing: 45, Speed: 20})
	require.NoError(t, err)
	require.Equal(t, 1, r.SegmentOrder, "segment must not switch outside halte radius without prior dwell")
}

func TestSegmentSwitchAllowedAfterDepartingStopRadius(t *testing.T) {
	line := threeStopLine()
	stops := threeStopRoute()
	cfg := snaptoline.RouteSnapConfig(stops, snaptoline.WithSegmentSwitchStopRadiusMeter(20))
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	mid, _ := snaptoline.PointAtMeasure(line, 15)
	_, err = snapper.Snap(snaptoline.GPSPoint{Point: mid, Bearing: 45, Speed: 20})
	require.NoError(t, err)

	// Inside halte B radius, past segment-1 destination — commits segment 2.
	pastB, _ := snaptoline.PointAtMeasure(line, 38)
	gate := stops[1].Point
	require.Less(t, snaptoline.DistanceMeter(pastB, gate), 20.0)
	atGate, err := snapper.Snap(snaptoline.GPSPoint{Point: pastB, Bearing: 45, Speed: 20})
	require.NoError(t, err)
	require.Equal(t, 2, atGate.SegmentOrder)

	// Off-route outside halte radius after departure — must stay on segment 2, not regress.
	farFromB := orb.Point{pastB[0] + 0.0005, pastB[1]}
	offRoute, err := snapper.Snap(snaptoline.GPSPoint{Point: farFromB, Bearing: 45, Speed: 20})
	require.NoError(t, err)
	require.Equal(t, 2, offRoute.SegmentOrder, "departure latch must keep forward segment after leaving halte")
}

func TestSegmentSwitchDepartLatchWithoutInRadiusSwitch(t *testing.T) {
	line := threeStopLine()
	stops := threeStopRoute()
	cfg := snaptoline.RouteSnapConfig(stops, snaptoline.WithSegmentSwitchStopRadiusMeter(20))
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	mid, _ := snaptoline.PointAtMeasure(line, 15)
	seed, err := snapper.Snap(snaptoline.GPSPoint{Point: mid, Bearing: 45, Speed: 20})
	require.NoError(t, err)
	require.Equal(t, 1, seed.SegmentOrder)

	// Dwell inside halte B but still on segment 1 (before destination measure).
	nearB, _ := snaptoline.PointAtMeasure(line, 28)
	require.Less(t, snaptoline.DistanceMeter(nearB, stops[1].Point), 20.0)
	atHalte, err := snapper.Snap(snaptoline.GPSPoint{Point: nearB, Bearing: 45, Speed: 5})
	require.NoError(t, err)
	require.Equal(t, 1, atHalte.SegmentOrder, "still on segment 1 while inside halte before passing stop")

	// Leave halte off-route past stop B — latch forward to segment 2 only.
	pastB, _ := snaptoline.PointAtMeasure(line, 40)
	farFromB := orb.Point{pastB[0] + 0.0005, pastB[1]}
	require.Greater(t, snaptoline.DistanceMeter(farFromB, stops[1].Point), 20.0)
	departed, err := snapper.Snap(snaptoline.GPSPoint{Point: farFromB, Bearing: 45, Speed: 20})
	require.NoError(t, err)
	require.Equal(t, 2, departed.SegmentOrder, "must latch to next segment after departing halte off-route")
}

func TestRouteSnapConfigEnablesNextStopSegmentGuard(t *testing.T) {
	cfg := snaptoline.RouteSnapConfig(threeStopRoute())
	require.True(t, cfg.RequireNextStopBeforeSegmentSwitch)
	require.True(t, cfg.RequireStopRadiusForSegmentSwitch)
	require.Equal(t, snaptoline.DefaultRouteSegmentSwitchStopRadiusMeter, cfg.SegmentSwitchStopRadiusMeter)
	require.Greater(t, cfg.NextStopPassToleranceMeter, 0.0)

	disabled := snaptoline.RouteSnapConfig(threeStopRoute(), snaptoline.WithRequireNextStopBeforeSegmentSwitch(false))
	require.False(t, disabled.RequireNextStopBeforeSegmentSwitch)
}
