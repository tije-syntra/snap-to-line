package snaptoline_test

import (
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestWildGPSJumpFreezesBackwardSnap(t *testing.T) {
	line := orb.LineString{
		{106.0, -6.0},
		{106.05, -6.0},
		{106.1, -6.0},
	}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
		{ID: "C", Order: 3, Point: line[2]},
	}

	cfg := snaptoline.RouteSnapConfig(stops,
		snaptoline.WithMeasureRegressionTolerance(10),
	)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := time.Now().UnixMilli()
	seed, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.04, -6.0},
		Speed:     30,
		Bearing:   90,
		Timestamp: ts,
	})
	require.NoError(t, err)
	seedM := snapper.RouteMeasure(seed.SegmentOrder, seed.Progress)

	// Wild backward raw GPS (~2.2 km along line) at low reported speed.
	wild, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.02, -6.0},
		Speed:     10,
		Bearing:   270,
		Timestamp: ts + 2000,
	})
	require.NoError(t, err)
	require.True(t, wild.HeldSegment)
	require.Contains(t, []string{"wild_gps_backward", "wild_gps_regression"}, wild.HeldReason)
	wildM := snapper.RouteMeasure(wild.SegmentOrder, wild.Progress)
	require.GreaterOrEqual(t, wildM, seedM-10)
}

func TestWildGPSJumpCapsForwardAdvance(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.1, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.RouteSnapConfig(stops,
		snaptoline.WithMeasureRegressionTolerance(10),
		snaptoline.WithMaxForwardSnapMeter(50),
	)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := time.Now().UnixMilli()
	seed, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.02, -6.0},
		Speed:     30,
		Timestamp: ts,
	})
	require.NoError(t, err)
	seedM := snapper.RouteMeasure(seed.SegmentOrder, seed.Progress)

	// Wild forward jump — capped when MaxForwardSnapMeter is set.
	wild, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.08, -6.0},
		Speed:     5,
		Timestamp: ts + 2000,
	})
	require.NoError(t, err)
	wildM := snapper.RouteMeasure(wild.SegmentOrder, wild.Progress)
	require.InDelta(t, 50.0, wildM-seedM, 2.0)
	require.Equal(t, "forward_cap", wild.HeldReason)
}
