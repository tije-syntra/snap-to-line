package snaptoline_test

import (
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestHoldLastSegmentOnNoCandidates(t *testing.T) {
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
	cfg.MaxSnapDistanceMeter = 28

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := time.Now().UnixMilli()
	first, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.025, -6.0},
		Speed:     30,
		Bearing:   90,
		Timestamp: ts,
	})
	require.NoError(t, err)
	require.Greater(t, first.SegmentOrder, 0)
	require.False(t, first.HeldSegment)

	// ~39 m north of the line — beyond max snap (28 m) but within hold radius (60 m).
	drifted, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.025, -6.00035},
		Speed:     30,
		Bearing:   90,
		Timestamp: ts + 2000,
	})
	require.NoError(t, err)
	require.True(t, drifted.HeldSegment)
	require.Equal(t, "no_candidates", drifted.HeldReason)
	require.Equal(t, first.SegmentOrder, drifted.SegmentOrder)
	require.Equal(t, first.SegmentID, drifted.SegmentID)
	require.False(t, drifted.IsOffRoute)
	require.GreaterOrEqual(t, drifted.Confidence, 0.25)
}

func TestHoldLastSegmentDisabled(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.1, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.RouteSnapConfig(stops, snaptoline.WithHoldLastSegmentOnMiss(false))
	cfg.MaxSnapDistanceMeter = 28

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	_, err = snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.05, -6.0}})
	require.NoError(t, err)

	result, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.05, -6.00035}})
	require.NoError(t, err)
	require.Empty(t, result.SegmentID)
	require.True(t, result.IsOffRoute)
}

func TestHoldLastSegmentExpiresAfterMaxAge(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.1, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.RouteSnapConfig(stops,
		snaptoline.WithHoldLastSegmentMaxAgeMs(5000),
	)
	cfg.MaxSnapDistanceMeter = 28

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := time.Now().UnixMilli()
	_, err = snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.05, -6.0},
		Timestamp: ts,
	})
	require.NoError(t, err)

	result, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.05, -6.00035},
		Timestamp: ts + 6000,
	})
	require.NoError(t, err)
	require.Empty(t, result.SegmentID)
	require.True(t, result.IsOffRoute)
}

func TestHoldLastSegmentFarBeyondNormalMaxDist(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.1, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.RouteSnapConfig(stops)
	cfg.MaxSnapDistanceMeter = 28

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := time.Now().UnixMilli()
	first, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.05, -6.0},
		Timestamp: ts,
	})
	require.NoError(t, err)
	require.NotEmpty(t, first.SegmentID)

	// ~80 m north — beyond normal max snap but hold keeps segment.
	drifted, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.05, -6.00072},
		Timestamp: ts + 3000,
	})
	require.NoError(t, err)
	require.True(t, drifted.HeldSegment)
	require.NotEmpty(t, drifted.SegmentID)
	require.Equal(t, first.SegmentID, drifted.SegmentID)
	require.Greater(t, drifted.DistanceMeter, 28.0)
	require.False(t, drifted.IsOffRoute)
}
