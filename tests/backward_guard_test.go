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

	cfg := snaptoline.RouteSnapConfig(stops, snaptoline.WithRequireStopRadiusForSegmentSwitch(false))
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

// Parallel approach lanes near terminal B (~5 m offset), similar to SOE Terminal 1B overlap.
// Segment 2 geometry folds through south lane then north centerline before the terminal.
func terminalParallelApproachLine() orb.LineString {
	return orb.LineString{
		{106.656100, -6.129000},
		{106.656250, -6.129150},
		{106.656260, -6.129180}, // south approach lane (correct driving path)
		{106.656310, -6.129220}, // north centerline (parallel branch on same segment)
		{106.656350, -6.129280}, // terminal B
		{106.656200, -6.129100},
	}
}

func terminalParallelApproachStops() []snaptoline.Stop {
	line := terminalParallelApproachLine()
	return []snaptoline.Stop{
		{ID: "S5", Name: "Before T1B", Order: 1, Point: line[0]},
		{ID: "S6", Name: "Approach", Order: 2, Point: line[1]},
		{ID: "B08326P", Name: "Terminal 1B", Order: 3, Point: line[4]},
		{ID: "S8", Name: "After T1B", Order: 4, Point: line[5]},
	}
}

func TestParallelApproachDoesNotJumpToOffsetLane(t *testing.T) {
	line := terminalParallelApproachLine()
	stops := terminalParallelApproachStops()
	cfg := snaptoline.RouteSnapConfig(stops)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	// Establish segment 2 on the south approach lane.
	_, err = snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.656310, -6.129220}, Bearing: 137, Speed: 11})
	require.NoError(t, err)
	south, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.656260, -6.129180}, Bearing: 137, Speed: 11})
	require.NoError(t, err)
	require.Equal(t, 2, south.SegmentOrder)

	// GPS biased toward the north parallel branch (TJ-611 screenshot coords).
	r, err := snapper.Snap(snaptoline.GPSPoint{
		Point:   orb.Point{106.656327, -6.129180},
		Bearing: 137,
		Speed:   11,
	})
	require.NoError(t, err)
	require.Equal(t, 2, r.SegmentOrder)

	jump := snaptoline.DistanceMeter(south.SnappedPoint, r.SnappedPoint)
	require.Less(t, jump, 4.0, "snap should not laterally jump to the parallel north branch")
}

func TestMeasureRegressionClampedRegardlessOfConfidence(t *testing.T) {
	line := terminalOverlapLine()
	stops := terminalOverlapStops()
	cfg := snaptoline.RouteSnapConfig(stops,
		snaptoline.WithMeasureRegressionTolerance(8),
		snaptoline.WithRequireStopRadiusForSegmentSwitch(false),
	)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	advance := []snaptoline.GPSPoint{
		{Point: orb.Point{106.654730, -6.129850}, Bearing: 180, Speed: 20},
		{Point: orb.Point{106.654760, -6.130050}, Bearing: 180, Speed: 20},
		{Point: orb.Point{106.654800, -6.130350}, Bearing: 180, Speed: 15},
	}
	var lastMeasure float64
	var lastOrder int
	for _, p := range advance {
		r, err := snapper.Snap(p)
		require.NoError(t, err)
		lastOrder = r.SegmentOrder
		lastMeasure = snapper.RouteMeasure(r.SegmentOrder, r.Progress)
	}
	require.Equal(t, 2, lastOrder)

	// Noisy point near folded return path — must not regress segment order or measure.
	r2, err := snapper.Snap(snaptoline.GPSPoint{
		Point: orb.Point{106.654714, -6.129853}, Bearing: 0, Speed: 15,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, r2.SegmentOrder, lastOrder)
	require.GreaterOrEqual(t, snapper.RouteMeasure(r2.SegmentOrder, r2.Progress), lastMeasure-8)
}

func TestStationaryGPSDoesNotJumpOnFoldedSegment(t *testing.T) {
	line := terminalParallelApproachLine()
	stops := terminalParallelApproachStops()
	cfg := snaptoline.RouteSnapConfig(stops)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	// Lock onto segment 2 south approach lane.
	_, err = snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.656310, -6.129220}, Bearing: 137, Speed: 11})
	require.NoError(t, err)
	seed, err := snapper.Snap(snaptoline.GPSPoint{
		Point: orb.Point{106.656260, -6.129180}, Bearing: 137, Speed: 11,
	})
	require.NoError(t, err)
	require.Equal(t, 2, seed.SegmentOrder)

	// Noisy stationary GPS biased toward the parallel north branch (TJ-613 scenario).
	noisy := []orb.Point{
		{106.656327, -6.129180},
		{106.656320, -6.129175},
		{106.656335, -6.129182},
		{106.656318, -6.129178},
		{106.656330, -6.129181},
	}
	var prevSnap orb.Point
	var havePrev bool
	for _, p := range noisy {
		r, err := snapper.Snap(snaptoline.GPSPoint{Point: p, Bearing: 96, Speed: 0})
		require.NoError(t, err)
		require.Equal(t, 2, r.SegmentOrder)
		if havePrev {
			jump := snaptoline.DistanceMeter(prevSnap, r.SnappedPoint)
			require.Less(t, jump, 4.0, "stationary snap should not hop between parallel branches")
		}
		prevSnap = r.SnappedPoint
		havePrev = true
	}
}

func TestPreferNearbyOnActiveSegmentWhenFarSnap(t *testing.T) {
	line := terminalParallelApproachLine()
	stops := terminalParallelApproachStops()
	cfg := snaptoline.RouteSnapConfig(stops, snaptoline.WithMeasureRegressionTolerance(10))
	cfg.MaxSnapDistanceMeter = 28
	cfg.MaxBearingDiffDegree = 40
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	_, err = snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.656310, -6.129220}, Bearing: 137, Speed: 11})
	require.NoError(t, err)
	seed, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.656260, -6.129180}, Bearing: 137, Speed: 11})
	require.NoError(t, err)
	require.Equal(t, 2, seed.SegmentOrder)

	// GPS biased toward parallel branch — snap should stay on segment 2 and stay close to raw.
	r, err := snapper.Snap(snaptoline.GPSPoint{
		Point: orb.Point{106.656327, -6.129180}, Bearing: 137, Speed: 11,
	})
	require.NoError(t, err)
	require.Equal(t, 2, r.SegmentOrder)
	require.Less(t, r.DistanceMeter, 12.0,
		"snap should stay near raw GPS on the active segment")
}

func TestClampAllowsForwardProgressAlongSegment(t *testing.T) {
	line := orb.LineString{
		{106.662000, -6.123000},
		{106.662100, -6.122900},
		{106.662200, -6.122800},
		{106.662300, -6.122700},
		{106.662400, -6.122600},
	}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[4]},
	}
	cfg := snaptoline.RouteSnapConfig(stops, snaptoline.WithMeasureRegressionTolerance(8))
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	p1, _ := snaptoline.PointAtMeasure(line, 12)
	p2, _ := snaptoline.PointAtMeasure(line, 28)

	seed, err := snapper.Snap(snaptoline.GPSPoint{Point: p1, Bearing: 45, Speed: 15})
	require.NoError(t, err)
	seedMeasure := snapper.RouteMeasure(seed.SegmentOrder, seed.Progress)

	r, err := snapper.Snap(snaptoline.GPSPoint{Point: p2, Bearing: 45, Speed: 15})
	require.NoError(t, err)
	nextMeasure := snapper.RouteMeasure(r.SegmentOrder, r.Progress)
	require.Greater(t, nextMeasure, seedMeasure,
		"snap measure should advance when GPS moves forward along segment")
}

func TestRouteSnapConfigEnablesBackwardGuard(t *testing.T) {
	stops := terminalOverlapStops()
	cfg := snaptoline.RouteSnapConfig(stops)
	require.True(t, cfg.PreventBackwardTransition)
	require.Greater(t, cfg.MeasureRegressionToleranceMeter, 0.0)
	require.Greater(t, cfg.ClampBackwardMinConfidence, 0.0)
	require.Greater(t, cfg.ClampDwellSpeedKmh, 0.0)
}
