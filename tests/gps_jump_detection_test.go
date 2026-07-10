package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestExpectedGpsDistanceUsesFloor(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.GpsJumpExpectedFactor = 1.5
	cfg.GpsJumpMinExpectedMeter = 75

	expected := snaptoline.ExpectedGpsDistanceM(
		cfg,
		snaptoline.GPSPoint{Point: orb.Point{106.0, -6.0}, Speed: 20, Timestamp: 4000},
		&snaptoline.GPSPoint{Point: orb.Point{106.0, -6.0}, Speed: 20, Timestamp: 1000},
		1000,
	)
	require.InDelta(t, 75, expected, 1)
}

func TestClassifyJumpRatioBands(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.GpsJumpWarningRatio = 1.5
	cfg.GpsJumpSuspiciousRatio = 3
	cfg.GpsJumpRejectRatio = 5

	require.Equal(t, snaptoline.GpsJumpLevelNormal, snaptoline.ClassifyJumpRatio(1.2, cfg))
	require.Equal(t, snaptoline.GpsJumpLevelWarning, snaptoline.ClassifyJumpRatio(2, cfg))
	require.Equal(t, snaptoline.GpsJumpLevelSuspicious, snaptoline.ClassifyJumpRatio(4, cfg))
	require.Equal(t, snaptoline.GpsJumpLevelReject, snaptoline.ClassifyJumpRatio(6, cfg))
}

func TestUpdateJumpCountThreshold(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.GpsJumpCountDistanceMeter = 150
	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)

	snaptoline.UpdateJumpCount(state, 200, cfg)
	require.Equal(t, 1, state.JumpCount)

	snaptoline.UpdateJumpCount(state, 100, cfg)
	require.Equal(t, 1, state.JumpCount)
}

func TestEvaluateGpsJumpRejectLevel(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.GpsJumpDetection = true
	cfg.GpsJumpExpectedFactor = 1.5
	cfg.GpsJumpMinExpectedMeter = 75
	cfg.GpsJumpRejectRatio = 5

	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)
	state.LastPoint = &snaptoline.GPSPoint{
		Point:     orb.Point{106.0, -6.0},
		Speed:     25,
		Timestamp: 1000,
	}
	state.LastTimestamp = 1000

	evaluation := snaptoline.EvaluateGpsJump(state, cfg, snaptoline.GPSPoint{
		Point:     orb.Point{106.0045, -6.0},
		Speed:     25,
		Timestamp: 4000,
	})

	require.NotNil(t, evaluation)
	require.Equal(t, snaptoline.GpsJumpLevelReject, evaluation.Level)
	require.Greater(t, evaluation.JumpRatio, 5.0)
	require.Equal(t, 1, state.JumpCount)
}

func TestSnapperHoldsOnGpsJumpReject(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.1, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.DefaultConfig()
	cfg.GpsJumpDetection = true
	cfg.GpsJumpExpectedFactor = 1.5
	cfg.GpsJumpMinExpectedMeter = 75
	cfg.GpsJumpRejectRatio = 5
	cfg.TeleportDetection = false

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := int64(1000)
	first, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.01, -6.0001},
		Speed:     25,
		Timestamp: ts,
	})
	require.NoError(t, err)
	require.Greater(t, first.SegmentOrder, 0)

	ts += 3000
	jumped, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.05, -6.0001},
		Speed:     25,
		Timestamp: ts,
	})
	require.NoError(t, err)
	require.True(t, jumped.HeldSegment)
	require.Equal(t, "gps_jump_reject", jumped.HeldReason)
	require.Equal(t, first.SegmentOrder, jumped.SegmentOrder)
	require.Equal(t, string(snaptoline.GpsJumpLevelReject), jumped.GpsJumpLevel)
}

func TestRouteSnapConfigEnablesGpsJumpDetection(t *testing.T) {
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: orb.Point{106.0, -6.0}},
		{ID: "B", Order: 2, Point: orb.Point{106.1, -6.0}},
	}
	cfg := snaptoline.RouteSnapConfig(stops)
	require.True(t, cfg.GpsJumpDetection)
	require.Equal(t, 75.0, snaptoline.ResolveGpsJumpMinExpectedMeter(cfg))
	require.Equal(t, 5.0, cfg.GpsJumpRejectRatio)
}
