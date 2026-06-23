package snaptoline_test

import (
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestForwardSnapCapAt50Meters(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.2, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.RouteSnapConfig(stops,
		snaptoline.WithMaxForwardSnapMeter(50),
	)
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := time.Now().UnixMilli()
	seed, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.02, -6.0},
		Speed:     40,
		Timestamp: ts,
	})
	require.NoError(t, err)
	seedM := snapper.RouteMeasure(seed.SegmentOrder, seed.Progress)

	// Raw GPS far ahead on the line (~8+ km); snap must not jump more than 50 m forward.
	far, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.10, -6.0},
		Speed:     40,
		Timestamp: ts + 2000,
	})
	require.NoError(t, err)
	farM := snapper.RouteMeasure(far.SegmentOrder, far.Progress)
	advance := farM - seedM
	require.InDelta(t, 50.0, advance, 2.0, "snap should cap forward advance to 50 m on the line")
	require.Equal(t, "forward_cap", far.HeldReason)
}

func TestForwardSnapCapDisabled(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.2, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.RouteSnapConfig(stops,
		snaptoline.WithMaxForwardSnapMeter(0),
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

	far, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.10, -6.0},
		Timestamp: ts + 2000,
	})
	require.NoError(t, err)
	farM := snapper.RouteMeasure(far.SegmentOrder, far.Progress)
	require.Greater(t, farM-seedM, 100.0)
}
