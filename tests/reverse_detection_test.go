package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func makeReverseSegment(order int, fromM, toM float64) snaptoline.Segment {
	return snaptoline.Segment{
		ID:          "seg-" + string(rune('0'+order)),
		FromStop:    snaptoline.Stop{ID: "s" + string(rune('0'+order)), Point: orb.Point{106.0, -6.0}, Order: order},
		ToStop:      snaptoline.Stop{ID: "s" + string(rune('1'+order)), Point: orb.Point{106.01, -6.0}, Order: order + 1},
		FromMeasure: fromM,
		ToMeasure:   toM,
		Geometry:    orb.LineString{{106.0, -6.0}, {106.01, -6.0}},
		Order:       order,
		Direction:   snaptoline.DirectionUnknown,
		Bearing:     90,
	}
}

func makeReverseCandidate(seg snaptoline.Segment, measure float64) snaptoline.Candidate {
	return snaptoline.Candidate{
		Segment:            seg,
		Measure:            measure,
		SnappedPoint:       seg.Geometry[0],
		DistanceMeter:      0,
		LineBearing:        90,
		EmissionScore:      1,
		DirectionScore:     1,
		TripDirectionScore: 1,
	}
}

func TestReverseResolvesDefaults(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	require.Equal(t, 2, snaptoline.ResolveReverseAcceptAfterSamples(cfg))
	require.Equal(t, 15.0, snaptoline.ResolveReverseIgnoreMeter(cfg))
	require.Equal(t, 30.0, snaptoline.ResolveReverseHoldMeter(cfg))
}

func TestUpdateReverseCount(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.ReverseMeasureEpsilonMeter = 0.5
	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)

	snaptoline.UpdateReverseCount(state, -10, cfg)
	require.Equal(t, 1, state.ReverseCount)

	snaptoline.UpdateReverseCount(state, -5, cfg)
	require.Equal(t, 2, state.ReverseCount)

	snaptoline.UpdateReverseCount(state, 2, cfg)
	require.Equal(t, 0, state.ReverseCount)
	require.False(t, state.TurnaroundValidated)
}

func TestReverseToleranceBands(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.ReverseDetection = true
	cfg.ReverseMeasureEpsilonMeter = 0.5
	cfg.ReverseIgnoreMeter = 15
	cfg.ReverseHoldMeter = 30
	cfg.ReverseWarningMeter = 50

	seg := makeReverseSegment(1, 0, 1000)
	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)
	state.LastBest = &snaptoline.Candidate{
		Segment:      seg,
		Measure:      500,
		SnappedPoint: seg.Geometry[0],
		LineBearing:  90,
	}

	point := snaptoline.GPSPoint{Point: orb.Point{106.0, -6.0}}
	require.Equal(t, snaptoline.ReverseActionIgnore, snaptoline.EvaluateReverseDetection(state, cfg, point, 490, nil).Action)
	require.Equal(t, snaptoline.ReverseActionHold, snaptoline.EvaluateReverseDetection(state, cfg, point, 475, nil).Action)
	require.Equal(t, snaptoline.ReverseActionWarning, snaptoline.EvaluateReverseDetection(state, cfg, point, 455, nil).Action)
	require.Equal(t, snaptoline.ReverseActionReverseCandidate, snaptoline.EvaluateReverseDetection(state, cfg, point, 430, nil).Action)
}

func TestReverseHoldUntilTurnaround(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.ReverseDetection = true
	cfg.ReverseMeasureEpsilonMeter = 0.5
	cfg.ReverseAcceptAfterSamples = 2
	cfg.ReverseHoldMeter = 30
	cfg.ReverseTurnDetection = true

	seg := makeReverseSegment(1, 0, 1000)
	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)
	state.LastBest = &snaptoline.Candidate{
		Segment:      seg,
		Measure:      500,
		SnappedPoint: seg.Geometry[0],
		LineBearing:  90,
	}
	state.ReverseCount = 2

	point := snaptoline.GPSPoint{Point: orb.Point{106.0, -6.0}}
	evaluation := snaptoline.EvaluateReverseDetection(state, cfg, point, 470, func() float64 { return 90 })
	require.Equal(t, snaptoline.ReverseActionHoldUntilTurnaround, evaluation.Action)

	line := orb.LineString{{106.0, -6.0}, {106.01, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}
	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	_, err = snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.005, -6.0}, Speed: 20})
	require.NoError(t, err)

	best := makeReverseCandidate(seg, 470)
	held, _ := snaptoline.StabilizeReverseDetection(snapper, point, &best, evaluation)
	require.NotNil(t, held)
	require.Equal(t, "reverse_turnaround_pending", held.HeldReason)
}

func TestBackwardSnapAllowedAfterTurnaround(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.ReverseDetection = true
	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)
	state.ReverseCount = 3
	state.TurnaroundValidated = true

	require.True(t, snaptoline.IsBackwardSnapAllowed(state, cfg))
}

func TestShouldWeakenDirectionAfterTurnaround(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.ReverseDetection = true
	require.True(t, snaptoline.ShouldWeakenDirectionValidation(
		snaptoline.GPSPoint{Point: orb.Point{106.0, -6.0}, Speed: 30},
		nil,
		cfg,
		true,
	))
}

func TestEvaluateTurnaroundFromMovementVectors(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.ReverseDetection = true
	cfg.ReverseTurnDetection = true
	cfg.ReverseTurnSampleWindow = 3
	cfg.ReverseTurnMinMovementMeter = 5
	cfg.ReverseTurnMinMovementAngleDegree = 90
	cfg.ReverseTurnCumulativeAngleDegree = 90
	cfg.ReverseTurnRouteOppositionDegree = 45
	cfg.ReverseMinSpeedKmh = 1

	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)
	snaptoline.PushRecentGpsPoint(state, snaptoline.GPSPoint{Point: orb.Point{106.0, -6.0}, Speed: 20, Timestamp: 1000}, 3)
	snaptoline.PushRecentGpsPoint(state, snaptoline.GPSPoint{Point: orb.Point{106.02, -6.0}, Speed: 20, Timestamp: 2000}, 3)
	p3 := snaptoline.GPSPoint{Point: orb.Point{106.015, -6.0}, Speed: 20, Timestamp: 3000}
	snaptoline.PushRecentGpsPoint(state, p3, 3)

	require.True(t, snaptoline.EvaluateTurnaround(state, cfg, p3, 90))
}
