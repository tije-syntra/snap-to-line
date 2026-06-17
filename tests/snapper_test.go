package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestSnapNormalToLinestring(t *testing.T) {
	line := orb.LineString{
		{106.0, -6.0},
		{106.01, -6.0},
		{106.02, -6.0},
	}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[2]},
	}

	cfg := snaptoline.DefaultConfig()
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	point := snaptoline.GPSPoint{Point: orb.Point{106.01, -6.0001}}
	result, err := snapper.Snap(point)
	require.NoError(t, err)
	require.False(t, result.IsOffRoute)
	require.Less(t, result.DistanceMeter, cfg.MaxSnapDistanceMeter)
	require.NotEmpty(t, result.SegmentID)
}

func TestSnapToNearestSegment(t *testing.T) {
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

	cfg := snaptoline.DefaultConfig()
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	result, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.075, -6.0002}})
	require.NoError(t, err)
	require.Equal(t, 2, result.SegmentOrder)
}

func TestFarGPSIsOffRoute(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.1, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.DefaultConfig()
	cfg.MaxSnapDistanceMeter = 30

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	result, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.05, -6.01}})
	require.NoError(t, err)
	require.True(t, result.IsOffRoute)
}

func TestViterbiDoesNotJumpFarSegment(t *testing.T) {
	line := orb.LineString{
		{106.0, -6.0},
		{106.02, -6.0},
		{106.04, -6.0},
		{106.06, -6.0},
	}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
		{ID: "C", Order: 3, Point: line[2]},
		{ID: "D", Order: 4, Point: line[3]},
	}

	cfg := snaptoline.DefaultConfig()
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	first, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.005, -6.0001}})
	require.NoError(t, err)
	require.Equal(t, 1, first.SegmentOrder)

	second, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.015, -6.0001}})
	require.NoError(t, err)
	require.LessOrEqual(t, second.SegmentOrder, 2)
}

func TestResetClearsViterbiState(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.1, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.DefaultConfig()
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	_, err = snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.05, -6.0}})
	require.NoError(t, err)

	snapper.Reset()
	result, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.01, -6.0}})
	require.NoError(t, err)
	require.NotNil(t, result)
}
