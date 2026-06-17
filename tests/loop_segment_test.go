package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func loopRouteLine() orb.LineString {
	return orb.LineString{
		{106.0, -6.0},
		{106.1, -6.0},
		{106.1, -6.1},
		{106.0, -6.0},
	}
}

func loopRouteStops() []snaptoline.Stop {
	return []snaptoline.Stop{
		{ID: "A", Order: 1, Point: orb.Point{106.0, -6.0}},
		{ID: "B", Order: 2, Point: orb.Point{106.1, -6.0}},
		{ID: "C", Order: 3, Point: orb.Point{106.1, -6.1}},
		{ID: "A", Order: 4, Point: orb.Point{106.0, -6.0}},
	}
}

func loopRouteConfig() snaptoline.Config {
	return snaptoline.Config{
		Looping:                   true,
		AllowSameStartEndStop:     true,
		LoopClosureToleranceMeter: 10,
		MaxSnapDistanceMeter:      60,
		CandidateLimit:            8,
	}
}

func TestLoopStartEndSameStopShouldCreateValidClosingSegment(t *testing.T) {
	line := loopRouteLine()
	stops := loopRouteStops()
	cfg := loopRouteConfig()

	projected, err := snaptoline.ProjectStopsSequential(line, stops, cfg)
	require.NoError(t, err)

	require.Equal(t, "A", projected[0].Stop.ID)
	require.Equal(t, "A", projected[3].Stop.ID)
	require.Equal(t, 1, projected[0].Occurrence)
	require.Equal(t, 2, projected[3].Occurrence)
	require.True(t, projected[3].IsLoopClosure)
	require.Greater(t, projected[3].Measure, projected[2].Measure)
}

func TestClosingSegmentShouldUseEndMeasureNotFirstStopMeasure(t *testing.T) {
	line := loopRouteLine()
	stops := loopRouteStops()
	cfg := loopRouteConfig()

	projected, err := snaptoline.ProjectStopsSequential(line, stops, cfg)
	require.NoError(t, err)

	segments, err := snaptoline.BuildSegmentsFromProjectedStops(line, projected, cfg)
	require.NoError(t, err)

	last := segments[len(segments)-1]
	require.Equal(t, "C", last.FromStop.ID)
	require.Equal(t, "A", last.ToStop.ID)
	require.True(t, last.IsLoopClosing)
	require.Greater(t, last.ToMeasure, last.FromMeasure)
	require.NotEmpty(t, last.Geometry)
	require.InDelta(t, snaptoline.LineLengthMeter(line), last.ToMeasure, 1.0)
}

func TestClosingSegmentDoesNotSliceToFirstOccurrence(t *testing.T) {
	line := loopRouteLine()
	stops := loopRouteStops()
	cfg := loopRouteConfig()

	projected, err := snaptoline.ProjectStopsSequential(line, stops, cfg)
	require.NoError(t, err)

	segments, err := snaptoline.BuildSegmentsFromProjectedStops(line, projected, cfg)
	require.NoError(t, err)

	last := segments[len(segments)-1]
	require.Greater(t, snaptoline.LineLengthMeter(last.Geometry), 100.0)
}

func TestLoopingRouteLastToFirstTransition(t *testing.T) {
	line := loopRouteLine()
	stops := loopRouteStops()
	cfg := loopRouteConfig()
	cfg.UseBearingValidation = false

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	nearEnd := orb.Point{106.02, -6.02}
	_, err = snapper.Snap(snaptoline.GPSPoint{Point: nearEnd})
	require.NoError(t, err)

	nearStart := orb.Point{106.01, -6.0}
	result, err := snapper.Snap(snaptoline.GPSPoint{Point: nearStart})
	require.NoError(t, err)
	require.False(t, result.IsOffRoute)
}
