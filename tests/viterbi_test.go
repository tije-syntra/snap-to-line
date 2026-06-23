package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

// Parallel outbound (north) and inbound (south) paths offset by ~20m.
func parallelRouteLine() orb.LineString {
	return orb.LineString{
		{106.00, -6.0000},
		{106.05, -6.0000},
		{106.10, -6.0000},
		{106.10, -6.0002},
		{106.05, -6.0002},
		{106.00, -6.0002},
	}
}

func parallelRouteStops() []snaptoline.Stop {
	line := parallelRouteLine()
	return []snaptoline.Stop{
		{ID: "O1", Order: 1, Point: line[0]},
		{ID: "O2", Order: 2, Point: line[2]},
		{ID: "I1", Order: 3, Point: line[3]},
		{ID: "I2", Order: 4, Point: line[5]},
	}
}

func parallelRouteConfig(trip snaptoline.DirectionType) snaptoline.Config {
	return snaptoline.Config{
		MaxSnapDistanceMeter: 60,
		CandidateLimit:       8,
		UseBearingValidation: true,
		MaxBearingDiffDegree: 60,
		UseMovementBearing:   true,
		MinMovementMeter:     5,
		UseTripDirection:     true,
		TripDirection:        trip,
		SegmentDirections: []snaptoline.DirectionType{
			snaptoline.DirectionOutbound,
			snaptoline.DirectionOutbound,
			snaptoline.DirectionInbound,
		},
	}
}

func TestOutboundBusDoesNotSnapToInboundPath(t *testing.T) {
	line := parallelRouteLine()
	stops := parallelRouteStops()
	cfg := parallelRouteConfig(snaptoline.DirectionOutbound)

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	// GPS closer to inbound path but moving east (outbound bearing).
	point := snaptoline.GPSPoint{
		Point:   orb.Point{106.04, -6.00015},
		Bearing: 90,
		Speed:   10,
	}

	result, err := snapper.Snap(point)
	require.NoError(t, err)
	require.Equal(t, snaptoline.DirectionOutbound, result.Direction)
	require.LessOrEqual(t, result.SegmentOrder, 2)
}

func TestInboundBusDoesNotSnapToOutboundPath(t *testing.T) {
	line := parallelRouteLine()
	stops := parallelRouteStops()
	cfg := parallelRouteConfig(snaptoline.DirectionInbound)

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	point := snaptoline.GPSPoint{
		Point:   orb.Point{106.04, -6.00005},
		Bearing: 270,
		Speed:   10,
	}

	result, err := snapper.Snap(point)
	require.NoError(t, err)
	require.Equal(t, snaptoline.DirectionInbound, result.Direction)
	require.GreaterOrEqual(t, result.SegmentOrder, 3)
}

func TestStoppedBusWeakensDirectionValidation(t *testing.T) {
	line := parallelRouteLine()
	stops := parallelRouteStops()
	cfg := parallelRouteConfig(snaptoline.DirectionOutbound)

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	for _, speed := range []float64{0, 0.5, 2.5} {
		snapper.Reset()
		slow := snaptoline.GPSPoint{
			Point: orb.Point{106.04, -6.0001},
			Speed: speed,
		}
		result, err := snapper.Snap(slow)
		require.NoError(t, err, "speed=%v", speed)
		require.False(t, result.IsOffRoute, "speed=%v", speed)
	}
}

func TestViterbiStaysStableWithNoisyGPS(t *testing.T) {
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

	orders := make([]int, 0, 5)
	points := []orb.Point{
		{106.005, -6.0003},
		{106.015, -6.0002},
		{106.025, -6.0004},
		{106.035, -6.0001},
		{106.045, -6.0003},
	}

	for _, p := range points {
		result, err := snapper.Snap(snaptoline.GPSPoint{Point: p, Bearing: 90})
		require.NoError(t, err)
		orders = append(orders, result.SegmentOrder)
	}

	for i := 1; i < len(orders); i++ {
		require.LessOrEqual(t, orders[i]-orders[i-1], +2)
	}
}
