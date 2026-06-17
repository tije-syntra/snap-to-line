package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestSliceLineByMeasure(t *testing.T) {
	line := orb.LineString{
		{106.0, -6.0},
		{106.01, -6.0},
		{106.02, -6.0},
	}

	total := snaptoline.LineLengthMeter(line)
	sliced, err := snaptoline.SliceLineByMeasure(line, 0, total)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(sliced), 2)
	require.InDelta(t, line[0][0], sliced[0][0], 0.0001)
}

func TestSliceByMeasureNotByCoordinates(t *testing.T) {
	line := orb.LineString{
		{106.0, -6.0},
		{106.05, -6.0},
		{106.1, -6.0},
		{106.0, -6.0},
	}

	fromMeasure := snaptoline.LineLengthMeter(line) * 0.6
	toMeasure := snaptoline.LineLengthMeter(line)

	sliced, err := snaptoline.SliceLineByMeasure(line, fromMeasure, toMeasure)
	require.NoError(t, err)
	require.Greater(t, snaptoline.LineLengthMeter(sliced), 0.0)
	require.Greater(t, toMeasure, fromMeasure)
}

func TestBuildSegmentsFromProjectedStops(t *testing.T) {
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
	projected, err := snaptoline.ProjectStopsSequential(line, stops, cfg)
	require.NoError(t, err)

	segments, err := snaptoline.BuildSegmentsFromProjectedStops(line, projected, cfg)
	require.NoError(t, err)
	require.Len(t, segments, 2)
	require.NotEmpty(t, segments[0].Geometry)
	require.Greater(t, segments[0].ToMeasure, segments[0].FromMeasure)
}

func TestProjectedStopMeasuresAreMonotonic(t *testing.T) {
	line := orb.LineString{
		{106.0, -6.0},
		{106.03, -6.0},
		{106.06, -6.0},
		{106.09, -6.0},
	}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
		{ID: "C", Order: 3, Point: line[2]},
		{ID: "D", Order: 4, Point: line[3]},
	}

	cfg := snaptoline.DefaultConfig()
	projected, err := snaptoline.ProjectStopsSequential(line, stops, cfg)
	require.NoError(t, err)

	for i := 1; i < len(projected); i++ {
		require.GreaterOrEqual(t, projected[i].Measure, projected[i-1].Measure)
	}
}
