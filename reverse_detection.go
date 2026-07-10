package snaptoline

import (
	"time"

	"github.com/paulmach/orb"
)

const (
	DefaultReverseMeasureEpsilonMeter        = 0.5
	DefaultReverseAcceptAfterSamples         = 2
	DefaultReverseIgnoreMeter                = 15.0
	DefaultReverseHoldMeter                  = 30.0
	DefaultReverseWarningMeter               = 50.0
	DefaultReverseMinSpeedKmh                = 5.0
	DefaultReverseTurnSampleWindow           = 3
	DefaultReverseTurnMinMovementMeter       = 8.0
	DefaultReverseTurnMinMovementAngleDegree = 120.0
	DefaultReverseTurnCumulativeAngleDegree  = 140.0
	DefaultReverseTurnRouteOppositionDegree  = 90.0
)

type RecentGpsPoint struct {
	Point     orb.Point
	Timestamp int64
	Speed     float64
}

type ReverseAction string

const (
	ReverseActionForward              ReverseAction = "forward"
	ReverseActionIgnore               ReverseAction = "ignore"
	ReverseActionHold                 ReverseAction = "hold"
	ReverseActionWarning              ReverseAction = "warning"
	ReverseActionReverseCandidate     ReverseAction = "reverse_candidate"
	ReverseActionHoldUntilTurnaround  ReverseAction = "hold_until_turnaround"
	ReverseActionAcceptBackward       ReverseAction = "accept_backward"
)

type ReverseEvaluation struct {
	Action           ReverseAction
	DeltaMeasure     float64
	BackwardDistance float64
	BackwardSample   bool
	IsTrueTurnaround bool
}

func resolveReverseMeasureEpsilonMeter(cfg Config) float64 {
	if cfg.ReverseMeasureEpsilonMeter > 0 {
		return cfg.ReverseMeasureEpsilonMeter
	}
	return DefaultReverseMeasureEpsilonMeter
}

func ResolveReverseAcceptAfterSamples(cfg Config) int {
	if cfg.ReverseAcceptAfterSamples > 0 {
		return cfg.ReverseAcceptAfterSamples
	}
	return DefaultReverseAcceptAfterSamples
}

func ResolveReverseIgnoreMeter(cfg Config) float64 {
	if cfg.ReverseIgnoreMeter > 0 {
		return cfg.ReverseIgnoreMeter
	}
	return DefaultReverseIgnoreMeter
}

func ResolveReverseHoldMeter(cfg Config) float64 {
	if cfg.ReverseHoldMeter > 0 {
		return cfg.ReverseHoldMeter
	}
	return DefaultReverseHoldMeter
}

func resolveReverseWarningMeter(cfg Config) float64 {
	if cfg.ReverseWarningMeter > 0 {
		return cfg.ReverseWarningMeter
	}
	return DefaultReverseWarningMeter
}

func resolveReverseMinSpeedKmh(cfg Config) float64 {
	if cfg.ReverseMinSpeedKmh > 0 {
		return cfg.ReverseMinSpeedKmh
	}
	return DefaultReverseMinSpeedKmh
}

func resolveReverseTurnSampleWindow(cfg Config) int {
	if cfg.ReverseTurnSampleWindow >= 2 {
		return cfg.ReverseTurnSampleWindow
	}
	return DefaultReverseTurnSampleWindow
}

func resolveReverseTurnMinMovementMeter(cfg Config) float64 {
	if cfg.ReverseTurnMinMovementMeter > 0 {
		return cfg.ReverseTurnMinMovementMeter
	}
	return DefaultReverseTurnMinMovementMeter
}

func resolveReverseTurnMinMovementAngleDegree(cfg Config) float64 {
	if cfg.ReverseTurnMinMovementAngleDegree > 0 {
		return cfg.ReverseTurnMinMovementAngleDegree
	}
	return DefaultReverseTurnMinMovementAngleDegree
}

func resolveReverseTurnCumulativeAngleDegree(cfg Config) float64 {
	if cfg.ReverseTurnCumulativeAngleDegree > 0 {
		return cfg.ReverseTurnCumulativeAngleDegree
	}
	return DefaultReverseTurnCumulativeAngleDegree
}

func resolveReverseTurnRouteOppositionDegree(cfg Config) float64 {
	if cfg.ReverseTurnRouteOppositionDegree > 0 {
		return cfg.ReverseTurnRouteOppositionDegree
	}
	return DefaultReverseTurnRouteOppositionDegree
}

func PushRecentGpsPoint(state *ViterbiState, point GPSPoint, window int) {
	ts := point.Timestamp
	if ts <= 0 {
		ts = time.Now().UnixMilli()
	}
	entry := RecentGpsPoint{
		Point:     point.Point,
		Timestamp: ts,
		Speed:     point.Speed,
	}
	state.RecentGpsPoints = append(state.RecentGpsPoints, entry)
	for len(state.RecentGpsPoints) > window {
		state.RecentGpsPoints = state.RecentGpsPoints[1:]
	}
}

func (s *Snapper) estimateRouteMeasureOnLastSegment(point GPSPoint) *float64 {
	ref := s.state.LastBest
	if ref == nil {
		return nil
	}
	seg := ref.Segment
	proj := ProjectPointOnLine(seg.Geometry, point.Point)
	m := seg.FromMeasure + proj.Measure
	return &m
}

func movementBearing(a, b RecentGpsPoint) float64 {
	return BearingBetween(a.Point, b.Point)
}

func movementDistance(a, b RecentGpsPoint) float64 {
	return DistanceMeter(a.Point, b.Point)
}

func pointAtRelativeIndex(points []RecentGpsPoint, relIdx int) *RecentGpsPoint {
	lenPts := len(points)
	if relIdx > 0 || -relIdx >= lenPts {
		return nil
	}
	idx := lenPts - 1 + relIdx
	if idx < 0 || idx >= lenPts {
		return nil
	}
	return &points[idx]
}

func movementVector(points []RecentGpsPoint, fromIdx, toIdx int) (bearing, dist float64) {
	from := pointAtRelativeIndex(points, fromIdx)
	to := pointAtRelativeIndex(points, toIdx)
	if from == nil || to == nil {
		return 0, 0
	}
	dist = movementDistance(*from, *to)
	if dist <= 0 {
		return 0, 0
	}
	return movementBearing(*from, *to), dist
}

func (s *Snapper) lineBearingAtLastValidSnap() float64 {
	ref := s.state.LastBest
	if ref == nil {
		ref = s.state.LastGood
	}
	if ref == nil {
		return 0
	}
	rel := ref.Measure - ref.Segment.FromMeasure
	return BearingAtMeasure(ref.Segment.Geometry, rel)
}

func (s *Snapper) evaluateTurnaround(point GPSPoint) bool {
	return EvaluateTurnaround(s.state, s.config, point, s.lineBearingAtLastValidSnap())
}

func EvaluateTurnaround(state *ViterbiState, cfg Config, point GPSPoint, routeBearing float64) bool {
	if !cfg.ReverseTurnDetection {
		return false
	}

	points := state.RecentGpsPoints
	window := resolveReverseTurnSampleWindow(cfg)
	if len(points) < window {
		return false
	}

	minMoveM := resolveReverseTurnMinMovementMeter(cfg)
	minSpeed := resolveReverseMinSpeedKmh(cfg)
	v1, d1 := movementVector(points, -2, -1)
	v2, d2 := movementVector(points, -1, 0)
	if d1 < minMoveM || d2 < minMoveM || point.Speed < minSpeed {
		return false
	}

	turnAngle := BearingDiff(v1, v2)
	cumTurn := BearingDiff(
		movementBearing(points[0], points[1]),
		movementBearing(points[len(points)-2], points[len(points)-1]),
	)
	routeOpposition := BearingDiff(v2, routeBearing)

	return turnAngle >= resolveReverseTurnMinMovementAngleDegree(cfg) &&
		cumTurn >= resolveReverseTurnCumulativeAngleDegree(cfg) &&
		routeOpposition >= resolveReverseTurnRouteOppositionDegree(cfg)
}

func UpdateReverseCount(state *ViterbiState, deltaMeasure float64, cfg Config) {
	epsilon := resolveReverseMeasureEpsilonMeter(cfg)
	if deltaMeasure < -epsilon {
		state.ReverseCount++
	} else if deltaMeasure > epsilon {
		state.ReverseCount = 0
		state.TurnaroundValidated = false
	}
}

func toleranceAction(backwardDistance float64, cfg Config) ReverseAction {
	if backwardDistance < ResolveReverseIgnoreMeter(cfg) {
		return ReverseActionIgnore
	}
	if backwardDistance < ResolveReverseHoldMeter(cfg) {
		return ReverseActionHold
	}
	if backwardDistance < resolveReverseWarningMeter(cfg) {
		return ReverseActionWarning
	}
	return ReverseActionReverseCandidate
}

func IsBackwardSnapAllowed(state *ViterbiState, cfg Config) bool {
	if !cfg.ReverseDetection {
		return false
	}
	return state.TurnaroundValidated &&
		state.ReverseCount > ResolveReverseAcceptAfterSamples(cfg)
}

func (s *Snapper) shouldAllowBackwardSnap() bool {
	return IsBackwardSnapAllowed(s.state, s.config)
}

func (s *Snapper) evaluateReverseDetection(point GPSPoint, currentRouteMeasure float64) *ReverseEvaluation {
	return EvaluateReverseDetection(s.state, s.config, point, currentRouteMeasure, func() float64 {
		return s.lineBearingAtLastValidSnap()
	})
}

func EvaluateReverseDetection(state *ViterbiState, cfg Config, point GPSPoint, currentRouteMeasure float64, routeBearingFn func() float64) *ReverseEvaluation {
	if !cfg.ReverseDetection {
		return nil
	}

	epsilon := resolveReverseMeasureEpsilonMeter(cfg)
	acceptAfter := ResolveReverseAcceptAfterSamples(cfg)
	lastMeasure := currentRouteMeasure
	if state.LastBest != nil {
		lastMeasure = state.LastBest.Measure
	}
	deltaMeasure := currentRouteMeasure - lastMeasure
	backwardSample := deltaMeasure < -epsilon
	backwardDistance := -deltaMeasure
	if backwardDistance < 0 {
		backwardDistance = 0
	}

	UpdateReverseCount(state, deltaMeasure, cfg)

	if state.ReverseCount > acceptAfter && cfg.ReverseTurnDetection {
		routeBearing := 0.0
		if routeBearingFn != nil {
			routeBearing = routeBearingFn()
		}
		if EvaluateTurnaround(state, cfg, point, routeBearing) {
			state.TurnaroundValidated = true
			return &ReverseEvaluation{
				Action:           ReverseActionAcceptBackward,
				DeltaMeasure:     deltaMeasure,
				BackwardDistance: backwardDistance,
				BackwardSample:   backwardSample,
				IsTrueTurnaround: true,
			}
		}
		return &ReverseEvaluation{
			Action:           ReverseActionHoldUntilTurnaround,
			DeltaMeasure:     deltaMeasure,
			BackwardDistance: backwardDistance,
			BackwardSample:   backwardSample,
			IsTrueTurnaround: false,
		}
	}

	if !backwardSample {
		return &ReverseEvaluation{
			Action:           ReverseActionForward,
			DeltaMeasure:     deltaMeasure,
			BackwardDistance: backwardDistance,
			BackwardSample:   backwardSample,
			IsTrueTurnaround: false,
		}
	}

	return &ReverseEvaluation{
		Action:           toleranceAction(backwardDistance, cfg),
		DeltaMeasure:     deltaMeasure,
		BackwardDistance: backwardDistance,
		BackwardSample:   backwardSample,
		IsTrueTurnaround: false,
	}
}

func (s *Snapper) prepareReverseDetection(point GPSPoint) {
	if !s.config.ReverseDetection {
		return
	}
	window := resolveReverseTurnSampleWindow(s.config)
	PushRecentGpsPoint(s.state, point, window)

	estimated := s.estimateRouteMeasureOnLastSegment(point)
	if estimated == nil {
		return
	}

	lastMeasure := *estimated
	if s.state.LastBest != nil {
		lastMeasure = s.state.LastBest.Measure
	}
	deltaMeasure := *estimated - lastMeasure
	epsilon := resolveReverseMeasureEpsilonMeter(s.config)
	acceptAfter := ResolveReverseAcceptAfterSamples(s.config)
	tentativeCount := s.state.ReverseCount
	if deltaMeasure < -epsilon {
		tentativeCount++
	}

	if tentativeCount > acceptAfter &&
		s.config.ReverseTurnDetection &&
		s.evaluateTurnaround(point) {
		s.state.TurnaroundValidated = true
	}
}

func StabilizeReverseDetection(s *Snapper, point GPSPoint, best *Candidate, evaluation *ReverseEvaluation) (*SnapResult, *Candidate) {
	return s.stabilizeReverseDetection(point, best, evaluation)
}

func (s *Snapper) stabilizeReverseDetection(point GPSPoint, best *Candidate, evaluation *ReverseEvaluation) (*SnapResult, *Candidate) {
	if evaluation == nil || !s.config.ReverseDetection || best == nil {
		return nil, nil
	}

	switch evaluation.Action {
	case ReverseActionAcceptBackward, ReverseActionForward, ReverseActionIgnore:
		return nil, nil
	}

	if s.shouldAllowBackwardSnap() {
		return nil, nil
	}

	ref := s.state.LastBest
	if ref == nil {
		ref = s.state.LastGood
	}
	if ref == nil {
		return nil, nil
	}

	reason := "reverse_hold"
	switch evaluation.Action {
	case ReverseActionWarning:
		reason = "reverse_warning"
	case ReverseActionReverseCandidate:
		reason = "reverse_candidate"
	case ReverseActionHoldUntilTurnaround:
		reason = "reverse_turnaround_pending"
	}

	return s.freezeAtRefCandidate(point, ref, reason)
}

func ApplyReverseResultMetadata(state *ViterbiState, cfg Config, result *SnapResult, evaluation *ReverseEvaluation) *SnapResult {
	if !cfg.ReverseDetection || result == nil {
		return result
	}

	out := *result
	out.ReverseCount = state.ReverseCount
	out.TurnaroundValidated = state.TurnaroundValidated

	if evaluation == nil {
		return &out
	}

	switch evaluation.Action {
	case ReverseActionWarning:
		out.Confidence = clampConfidence(out.Confidence * 0.85)
	case ReverseActionReverseCandidate, ReverseActionHoldUntilTurnaround:
		out.Confidence = clampConfidence(out.Confidence * 0.75)
	}

	return &out
}
