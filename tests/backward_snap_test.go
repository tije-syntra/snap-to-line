package snaptoline_test

import (
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestNoBackwardSnapFreezesMeasureRegression(t *testing.T) {
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
		snaptoline.WithTeleportDetection(false),
		snaptoline.WithGpsJumpDetection(false),
		snaptoline.WithReverseDetection(false),
	)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := time.Now().UnixMilli()
	seed, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.04, -6.0},
		Speed:     30,
		Timestamp: ts,
	})
	require.NoError(t, err)
	seedM := snapper.RouteMeasure(seed.SegmentOrder, seed.Progress)

	// GPS behind previous position on the line.
	back, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.02, -6.0},
		Speed:     30,
		Timestamp: ts + 2000,
	})
	require.NoError(t, err)
	require.True(t, back.HeldSegment)
	backM := snapper.RouteMeasure(back.SegmentOrder, back.Progress)
	require.InDelta(t, seedM, backM, 1.0, "bus must not move backward on the route")
	require.GreaterOrEqual(t, back.SegmentOrder, seed.SegmentOrder)
}

func TestNoBackwardSnapAllowsForward(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.1, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.RouteSnapConfig(stops,
		snaptoline.WithTeleportDetection(false),
		snaptoline.WithGpsJumpDetection(false),
		snaptoline.WithReverseDetection(false),
	)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := time.Now().UnixMilli()
	seed, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.02, -6.0},
		Timestamp: ts,
	})
	require.NoError(t, err)
	seedM := snapper.RouteMeasure(seed.SegmentOrder, seed.Progress)

	ahead, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.025, -6.0},
		Speed:     30,
		Timestamp: ts + 2000,
	})
	require.NoError(t, err)
	aheadM := snapper.RouteMeasure(ahead.SegmentOrder, ahead.Progress)
	require.Greater(t, aheadM, seedM)
}

func TestNoBackwardCreepsForwardWhenGPSDriftsOffLine(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.2, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.RouteSnapConfig(stops,
		snaptoline.WithTeleportDetection(false),
		snaptoline.WithGpsJumpDetection(false),
		snaptoline.WithReverseDetection(false),
	)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := time.Now().UnixMilli()
	seed, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.08, -6.0},
		Speed:     35,
		Bearing:   90,
		Timestamp: ts,
	})
	require.NoError(t, err)
	seedM := snapper.RouteMeasure(seed.SegmentOrder, seed.Progress)

	var lastM float64
	for i := 1; i <= 6; i++ {
		lng := 106.08 + float64(i)*0.00015
		lat := -6.0 + float64(i)*0.00008 // drift off the line
		r, err := snapper.Snap(snaptoline.GPSPoint{
			Point:     orb.Point{lng, lat},
			Speed:     35,
			Bearing:   90,
			Timestamp: ts + int64(i)*2000,
		})
		require.NoError(t, err)
		lastM = snapper.RouteMeasure(r.SegmentOrder, r.Progress)
		require.GreaterOrEqual(t, lastM, seedM-1.0)
	}
	require.Greater(t, lastM, seedM+5.0, "bus should creep forward while raw GPS drifts off line")
}
