package snaptoline_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestBearingDiffOppositeDirection(t *testing.T) {
	diff := snaptoline.BearingDiff(90, 270)
	require.InDelta(t, 180, diff, 0.001)
}

func TestBearingDiffWrapAround(t *testing.T) {
	diff := snaptoline.BearingDiff(350, 10)
	require.InDelta(t, 20, diff, 0.001)
}

func TestDirectionScorePenalizesLargeDiff(t *testing.T) {
	score := snaptoline.DirectionScore(90, 60)
	require.Equal(t, 0.05, score)
}

func TestDirectionScoreAcceptsSmallDiff(t *testing.T) {
	score := snaptoline.DirectionScore(20, 60)
	require.InDelta(t, 1-(20.0/60.0), score, 0.001)
}

func TestTripDirectionScorePenalizesMismatch(t *testing.T) {
	score := snaptoline.TripDirectionScore(snaptoline.DirectionInbound, snaptoline.DirectionOutbound)
	require.Equal(t, 0.05, score)
}

func TestTripDirectionScoreAcceptsMatch(t *testing.T) {
	score := snaptoline.TripDirectionScore(snaptoline.DirectionOutbound, snaptoline.DirectionOutbound)
	require.Equal(t, 1.0, score)
}

func TestTransitionScoreLoopWrap(t *testing.T) {
	score := snaptoline.TransitionScore(3, 1, 3, true)
	require.Equal(t, 0.95, score)
}

func TestTransitionScoreBackwardIsLow(t *testing.T) {
	score := snaptoline.TransitionScore(3, 1, 4, false)
	require.Equal(t, 0.05, score)
}
