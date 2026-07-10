package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func makeSeqSegment(order int, fromM, toM float64) snaptoline.Segment {
	lng := 106.0 + float64(order-1)*0.02
	nextLng := lng + 0.02
	return snaptoline.Segment{
		ID:          "seg-" + string(rune('0'+order)),
		FromStop:    snaptoline.Stop{ID: "s" + string(rune('0'+order)), Point: orb.Point{lng, -6.0}, Order: order},
		ToStop:      snaptoline.Stop{ID: "s" + string(rune('1'+order)), Point: orb.Point{nextLng, -6.0}, Order: order + 1},
		FromMeasure: fromM,
		ToMeasure:   toM,
		Geometry:    orb.LineString{{lng, -6.0}, {nextLng, -6.0}},
		Order:       order,
		Direction:   snaptoline.DirectionUnknown,
		Bearing:     90,
	}
}

func TestComputeSegmentOrderDelta(t *testing.T) {
	require.Equal(t, 2, snaptoline.ComputeSegmentOrderDelta(2, 4, 4, false))
	require.Equal(t, 1, snaptoline.ComputeSegmentOrderDelta(2, 3, 4, false))
	require.Equal(t, 1, snaptoline.ComputeSegmentOrderDelta(4, 1, 4, true))
}

func TestRejectSegmentJumpCandidate(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.SegmentSequenceValidation = true
	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)
	seg2 := makeSeqSegment(2, 100, 200)
	seg4 := makeSeqSegment(4, 300, 400)
	state.LastValidSegmentOrder = 2
	state.LastBest = &snaptoline.Candidate{Segment: seg2, Measure: 150}

	jumped := snaptoline.Candidate{Segment: seg4, Measure: 350}
	require.True(t, snaptoline.RejectSegmentJumpCandidate(state, jumped, 4, cfg))

	next := snaptoline.Candidate{Segment: makeSeqSegment(3, 200, 300), Measure: 250}
	require.False(t, snaptoline.RejectSegmentJumpCandidate(state, next, 4, cfg))
}

func TestSequentialSegmentProgression(t *testing.T) {
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

	cfg := snaptoline.RouteSnapConfig(stops)
	cfg.SegmentSequenceValidation = true
	cfg.RequireNextStopBeforeSegmentSwitch = false
	cfg.RequireStopRadiusForSegmentSwitch = false

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	first, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.005, -6.0001}})
	require.NoError(t, err)
	require.Equal(t, 1, first.SegmentOrder)

	second, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.025, -6.0001}})
	require.NoError(t, err)
	require.Equal(t, 2, second.SegmentOrder)

	third, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.045, -6.0001}})
	require.NoError(t, err)
	require.Equal(t, 3, third.SegmentOrder)
}

func TestSnapperHoldsSegmentJumpTwoToFour(t *testing.T) {
	line := orb.LineString{
		{106.0, -6.0},
		{106.015, -6.0},
		{106.03, -6.0},
		{106.045, -6.0},
		{106.06, -6.0},
	}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
		{ID: "C", Order: 3, Point: line[2]},
		{ID: "D", Order: 4, Point: line[3]},
		{ID: "E", Order: 5, Point: line[4]},
	}

	cfg := snaptoline.RouteSnapConfig(stops)
	cfg.SegmentSequenceValidation = true
	cfg.RequireNextStopBeforeSegmentSwitch = false
	cfg.RequireStopRadiusForSegmentSwitch = false
	cfg.ReverseDetection = false
	cfg.GpsJumpDetection = false
	cfg.TeleportDetection = false
	cfg.SnapDistanceResetOnGrow = false

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	_, err = snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.005, -6.0001}})
	require.NoError(t, err)
	onSeg2, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.022, -6.0001}})
	require.NoError(t, err)
	require.Equal(t, 2, onSeg2.SegmentOrder)

	jumped, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.052, -6.0001}})
	require.NoError(t, err)
	require.Equal(t, 2, jumped.SegmentOrder)
	require.Equal(t, "segment_jump_not_allowed", jumped.HeldReason)
	require.Equal(t, "skipped_segment_order", jumped.RejectedReason)
	require.Equal(t, 1, jumped.SegmentJumpCount)
}
