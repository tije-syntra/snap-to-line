package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func stopContextLine() orb.LineString {
	return orb.LineString{
		{106.0, -6.0},
		{106.01, -6.0},
		{106.02, -6.0},
		{106.03, -6.0},
	}
}

func stopContextStops() []snaptoline.Stop {
	line := stopContextLine()
	return []snaptoline.Stop{
		{ID: "S0", Name: "Origin", Order: 0, Point: line[0]},
		{ID: "S1", Name: "Stop 1", Order: 1, Point: line[1]},
		{ID: "S2", Name: "Stop 2", Order: 2, Point: line[2]},
		{ID: "S9", Name: "Terminal", Order: 9, Point: line[3]},
	}
}

func stopContextCfg() snaptoline.Config {
	cfg := snaptoline.RouteSnapConfig(stopContextStops())
	cfg.SegmentSwitchStopRadiusMeter = 50
	return cfg
}

func TestStopContextEnRoute(t *testing.T) {
	line := stopContextLine()
	stops := stopContextStops()
	cfg := stopContextCfg()
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	mid, _ := snaptoline.PointAtMeasure(line, 1600)
	result, err := snapper.Snap(snaptoline.GPSPoint{
		Point:   mid,
		Bearing: 90,
		Speed:   35,
	})
	require.NoError(t, err)
	require.Empty(t, result.CurrStopName)
	require.Equal(t, "Stop 1", result.PrevStopName)
	require.Equal(t, "S1", result.PrevStopID)
	require.Equal(t, "Stop 2", result.NextStopName)
	require.Equal(t, "S2", result.NextStopID)
}

func TestStopContextArrived(t *testing.T) {
	line := stopContextLine()
	stops := stopContextStops()
	cfg := stopContextCfg()
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	result, err := snapper.Snap(snaptoline.GPSPoint{
		Point:   stops[2].Point,
		Bearing: 90,
		Speed:   5,
	})
	require.NoError(t, err)
	require.Equal(t, "Stop 1", result.PrevStopName)
	require.Equal(t, "Stop 2", result.CurrStopName)
	require.Equal(t, "S2", result.CurrStopID)
	require.Equal(t, "Terminal", result.NextStopName)
	require.Equal(t, "S9", result.NextStopID)
}

func TestStopContextDepartedFromHalte(t *testing.T) {
	line := stopContextLine()
	stops := stopContextStops()
	cfg := stopContextCfg()
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	_, err = snapper.Snap(snaptoline.GPSPoint{Point: stops[2].Point, Bearing: 90, Speed: 0})
	require.NoError(t, err)

	departed, err := snapper.Snap(snaptoline.GPSPoint{
		Point:   orb.Point{106.021, -6.0},
		Bearing: 90,
		Speed:   35,
	})
	require.NoError(t, err)
	require.Empty(t, departed.CurrStopName)
	require.Equal(t, "Stop 2", departed.PrevStopName)
	require.Equal(t, "S2", departed.PrevStopID)
	require.Equal(t, "Terminal", departed.NextStopName)
}

func TestStopContextArrivedAtTerminal(t *testing.T) {
	line := stopContextLine()
	stops := stopContextStops()
	cfg := stopContextCfg()
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	result, err := snapper.Snap(snaptoline.GPSPoint{
		Point:   stops[3].Point,
		Bearing: 90,
		Speed:   0,
	})
	require.NoError(t, err)
	require.Equal(t, "Stop 2", result.PrevStopName)
	require.Equal(t, "Terminal", result.CurrStopName)
	require.Empty(t, result.NextStopName)
}
