package snaptoline

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"
)

func TestRouteSnapConfigDefaults(t *testing.T) {
	stops := terminalOverlapStops()
	cfg := RouteSnapConfig(stops)

	require.True(t, cfg.PreventBackwardTransition)
	require.Equal(t, DefaultRouteMeasureRegressionToleranceMeter, cfg.MeasureRegressionToleranceMeter)
	require.Equal(t, DefaultRouteClampBackwardMinConfidence, cfg.ClampBackwardMinConfidence)
	require.Equal(t, DefaultRouteClampDwellSpeedKmh, cfg.ClampDwellSpeedKmh)
}

func TestRouteSnapConfigWithOptions(t *testing.T) {
	stops := terminalOverlapStops()
	cfg := RouteSnapConfig(stops,
		WithMeasureRegressionTolerance(45),
		WithClampDwellSpeedKmh(5),
		WithLooping(true),
	)

	require.Equal(t, 45.0, cfg.MeasureRegressionToleranceMeter)
	require.Equal(t, 5.0, cfg.ClampDwellSpeedKmh)
	require.True(t, cfg.Looping)
	require.Equal(t, DefaultRouteClampBackwardMinConfidence, cfg.ClampBackwardMinConfidence)
}

func TestRouteSnapConfigWithParamsStruct(t *testing.T) {
	stops := terminalOverlapStops()
	noBackward := false
	cfg := RouteSnapConfig(stops, RouteSnapParamsOption(RouteSnapParams{
		PreventBackwardTransition: &noBackward,
	}))

	require.False(t, cfg.PreventBackwardTransition)
}

func TestRouteSnapConfigDisableClamp(t *testing.T) {
	stops := terminalOverlapStops()
	cfg := RouteSnapConfig(stops, DisableBackwardClamp())
	require.Equal(t, 0.0, cfg.ClampBackwardMinConfidence)
}

func TestMergeRouteSnapParams(t *testing.T) {
	a := 10.0
	merged := mergeRouteSnapParams(RouteSnapParams{}, RouteSnapParams{
		MeasureRegressionToleranceMeter: &a,
	})
	require.NotNil(t, merged.MeasureRegressionToleranceMeter)
	require.Equal(t, 10.0, *merged.MeasureRegressionToleranceMeter)
}

func terminalOverlapStops() []Stop {
	line := terminalOverlapLine()
	return []Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
		{ID: "A2", Order: 3, Point: line[2]},
	}
}

func terminalOverlapLine() orb.LineString {
	return orb.LineString{
		{106.654700, -6.129700},
		{106.654760, -6.130080},
		{106.654820, -6.130450},
	}
}
