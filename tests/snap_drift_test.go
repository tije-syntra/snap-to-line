package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestGpsDriftResnapDoesNotRegressSegmentOrder(t *testing.T) {
	line := terminalParallelApproachLine()
	stops := terminalParallelApproachStops()
	cfg := snaptoline.RouteSnapConfig(stops)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	advance := []snaptoline.GPSPoint{
		{Point: orb.Point{106.656310, -6.129220}, Bearing: 137, Speed: 15},
		{Point: orb.Point{106.656260, -6.129180}, Bearing: 137, Speed: 15},
	}
	var lastOrder int
	var lastMeasure float64
	for _, p := range advance {
		r, err := snapper.Snap(p)
		require.NoError(t, err)
		lastOrder = r.SegmentOrder
		lastMeasure = snapper.RouteMeasure(r.SegmentOrder, r.Progress)
	}
	require.Equal(t, 2, lastOrder)

	// Noisy GPS biased toward folded geometry — must not regress segment order or measure.
	drift := []snaptoline.GPSPoint{
		{Point: orb.Point{106.656327, -6.129180}, Bearing: 137, Speed: 12},
		{Point: orb.Point{106.656340, -6.129170}, Bearing: 137, Speed: 12},
		{Point: orb.Point{106.656350, -6.129160}, Bearing: 137, Speed: 12},
	}
	for _, p := range drift {
		r, err := snapper.Snap(p)
		require.NoError(t, err)
		require.GreaterOrEqual(t, r.SegmentOrder, lastOrder)
		m := snapper.RouteMeasure(r.SegmentOrder, r.Progress)
		require.GreaterOrEqual(t, m, lastMeasure-10)
		lastOrder = r.SegmentOrder
		if m > lastMeasure {
			lastMeasure = m
		}
	}
}

func TestGpsDriftFarFromSnapDoesNotFreezeMeasure(t *testing.T) {
	line := orb.LineString{
		{106.655200, -6.129700},
		{106.655500, -6.129400},
		{106.655800, -6.129100},
		{106.656100, -6.128800},
		{106.656400, -6.128500},
		{106.656200, -6.129100}, // folded return
		{106.655900, -6.129400},
	}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[2]},
		{ID: "C", Order: 3, Point: line[4]},
	}

	cfg := snaptoline.RouteSnapConfig(stops,
		snaptoline.WithSnapDistanceResetOnGrow(false),
		snaptoline.WithMeasureRegressionTolerance(10),
	)
	cfg.MaxSnapDistanceMeter = 28

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	seed, err := snapper.Snap(snaptoline.GPSPoint{
		Point: orb.Point{106.656100, -6.128800}, Bearing: 45, Speed: 20,
	})
	require.NoError(t, err)
	seedMeasure := snapper.RouteMeasure(seed.SegmentOrder, seed.Progress)

	// GPS drifts far from snap while still moving along corridor — measure must not stay frozen.
	var prevMeasure float64
	stuckTicks := 0
	driftPath := []orb.Point{
		{106.656250, -6.128650},
		{106.656350, -6.128550},
		{106.656450, -6.128450},
		{106.656550, -6.128350},
		{106.656650, -6.128250},
	}
	for _, p := range driftPath {
		r, err := snapper.Snap(snaptoline.GPSPoint{Point: p, Bearing: 45, Speed: 18})
		require.NoError(t, err)
		m := snapper.RouteMeasure(r.SegmentOrder, r.Progress)
		if m <= prevMeasure+0.1 && r.DistanceMeter > 25 {
			stuckTicks++
		} else {
			stuckTicks = 0
		}
		prevMeasure = m
	}
	require.Less(t, stuckTicks, 3, "snap should resnap or advance when GPS drifts far from frozen position")
	require.Greater(t, prevMeasure, seedMeasure-10)
}

func TestPickNearestForwardCandidateSkipsLowerSegment(t *testing.T) {
	line := terminalOverlapLine()
	stops := terminalOverlapStops()
	cfg := snaptoline.RouteSnapConfig(stops)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	advance := []snaptoline.GPSPoint{
		{Point: orb.Point{106.654730, -6.129850}, Bearing: 180, Speed: 20},
		{Point: orb.Point{106.654760, -6.130050}, Bearing: 180, Speed: 20},
		{Point: orb.Point{106.654800, -6.130350}, Bearing: 180, Speed: 15},
	}
	for _, p := range advance {
		_, err := snapper.Snap(p)
		require.NoError(t, err)
	}

	r, err := snapper.Snap(snaptoline.GPSPoint{
		Point: orb.Point{106.654714, -6.129853}, Bearing: 0, Speed: 5,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, r.SegmentOrder, 2)
}
