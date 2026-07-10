package snaptoline

const (
	DefaultSegmentJumpRecoverySamples               = 3
	DefaultSegmentJumpRecoveryMinConfidence         = 0.8
	DefaultSegmentJumpRecoveryMaxDistanceMeter      = 30.0
	DefaultSegmentJumpRecoveryMaxDirectionDiffDegree = 45.0
	DefaultSegmentJumpRecoveryGpsFactor             = 1.5
)

type SegmentSequenceAction string

const (
	SegmentSequenceActionAccept      SegmentSequenceAction = "accept"
	SegmentSequenceActionSameSegment SegmentSequenceAction = "same_segment"
	SegmentSequenceActionNextSegment SegmentSequenceAction = "next_segment"
	SegmentSequenceActionHoldJump    SegmentSequenceAction = "hold_jump"
	SegmentSequenceActionHoldReverse SegmentSequenceAction = "hold_reverse"
	SegmentSequenceActionRecovery    SegmentSequenceAction = "recovery"
)

type SegmentSequenceEvaluation struct {
	Action       SegmentSequenceAction
	SegmentDelta int
	Accept       bool
	Hold         bool
}

func resolveSegmentJumpRecoverySamples(cfg Config) int {
	if cfg.SegmentJumpRecoverySamples > 0 {
		return cfg.SegmentJumpRecoverySamples
	}
	return DefaultSegmentJumpRecoverySamples
}

func resolveSegmentJumpRecoveryMinConfidence(cfg Config) float64 {
	if cfg.SegmentJumpRecoveryMinConfidence > 0 {
		return cfg.SegmentJumpRecoveryMinConfidence
	}
	return DefaultSegmentJumpRecoveryMinConfidence
}

func resolveSegmentJumpRecoveryMaxDistanceMeter(cfg Config) float64 {
	if cfg.SegmentJumpRecoveryMaxDistanceMeter > 0 {
		return cfg.SegmentJumpRecoveryMaxDistanceMeter
	}
	return DefaultSegmentJumpRecoveryMaxDistanceMeter
}

func resolveSegmentJumpRecoveryMaxDirectionDiffDegree(cfg Config) float64 {
	if cfg.SegmentJumpRecoveryMaxDirectionDiffDegree > 0 {
		return cfg.SegmentJumpRecoveryMaxDirectionDiffDegree
	}
	return DefaultSegmentJumpRecoveryMaxDirectionDiffDegree
}

func resolveSegmentJumpRecoveryGpsFactor(cfg Config) float64 {
	if cfg.SegmentJumpRecoveryGpsFactor > 0 {
		return cfg.SegmentJumpRecoveryGpsFactor
	}
	return DefaultSegmentJumpRecoveryGpsFactor
}

func ResolveLastValidSegmentOrder(state *ViterbiState) int {
	if state.LastValidSegmentOrder > 0 {
		return state.LastValidSegmentOrder
	}
	if state.LastBest != nil {
		return state.LastBest.Segment.Order
	}
	return 0
}

func ComputeSegmentOrderDelta(fromOrder, toOrder, segmentCount int, looping bool) int {
	if fromOrder <= 0 || toOrder <= 0 {
		return 0
	}
	if isLoopWrapTransition(fromOrder, toOrder, segmentCount, looping) {
		return 1
	}
	return toOrder - fromOrder
}

func (s *Snapper) segmentByOrder(order int) *Segment {
	for i := range s.segments {
		if s.segments[i].Order == order {
			return &s.segments[i]
		}
	}
	return nil
}

func (s *Snapper) hasValidCandidateForSkippedSegments(point GPSPoint, fromOrder, toOrder int) bool {
	for order := fromOrder + 1; order < toOrder; order++ {
		seg := s.segmentByOrder(order)
		if seg == nil {
			continue
		}
		proj := ProjectPointOnLine(seg.Geometry, point.Point)
		if proj.DistanceMeter <= s.config.MaxSnapDistanceMeter {
			return true
		}
	}
	return false
}

func (s *Snapper) gpsMovementPlausibleForRecovery(point GPSPoint) bool {
	if s.state.LastPoint == nil {
		return true
	}
	gpsDistance := DistanceMeter(s.state.LastPoint.Point, point.Point)
	expected := expectedGpsDistanceM(s.config, point, s.state.LastPoint, s.state.LastTimestamp)
	factor := resolveSegmentJumpRecoveryGpsFactor(s.config)
	return gpsDistance <= expected*factor
}

func (s *Snapper) canRecoverSegmentJump(point GPSPoint, result *SnapResult, best *Candidate) bool {
	cfg := s.config
	fromOrder := ResolveLastValidSegmentOrder(s.state)
	toOrder := best.Segment.Order
	if toOrder <= fromOrder+1 {
		return false
	}
	if s.state.SegmentJumpCount < resolveSegmentJumpRecoverySamples(cfg) {
		return false
	}
	if result.Confidence <= resolveSegmentJumpRecoveryMinConfidence(cfg) {
		return false
	}
	if result.DistanceMeter >= resolveSegmentJumpRecoveryMaxDistanceMeter(cfg) {
		return false
	}
	if result.DirectionDiff >= resolveSegmentJumpRecoveryMaxDirectionDiffDegree(cfg) {
		return false
	}
	if !s.gpsMovementPlausibleForRecovery(point) {
		return false
	}
	if s.hasValidCandidateForSkippedSegments(point, fromOrder, toOrder) {
		return false
	}
	return true
}

func EvaluateSegmentSequence(s *Snapper, point GPSPoint, result *SnapResult, best *Candidate) *SegmentSequenceEvaluation {
	return s.evaluateSegmentSequence(point, result, best)
}

func StabilizeSegmentSequence(s *Snapper, point GPSPoint, best *Candidate, result *SnapResult, evaluation *SegmentSequenceEvaluation) (*SnapResult, *Candidate) {
	return s.stabilizeSegmentSequence(point, best, result, evaluation)
}

func (s *Snapper) evaluateSegmentSequence(point GPSPoint, result *SnapResult, best *Candidate) *SegmentSequenceEvaluation {
	if best == nil || !s.config.SegmentSequenceValidation {
		return nil
	}

	fromOrder := ResolveLastValidSegmentOrder(s.state)
	toOrder := best.Segment.Order
	if fromOrder <= 0 {
		return &SegmentSequenceEvaluation{
			Action:       SegmentSequenceActionAccept,
			SegmentDelta: 0,
			Accept:       true,
			Hold:         false,
		}
	}

	delta := ComputeSegmentOrderDelta(fromOrder, toOrder, len(s.segments), s.config.Looping)

	if delta == 0 {
		s.state.SegmentJumpCount = 0
		return &SegmentSequenceEvaluation{
			Action:       SegmentSequenceActionSameSegment,
			SegmentDelta: delta,
			Accept:       true,
			Hold:         false,
		}
	}

	if delta == 1 {
		s.state.SegmentJumpCount = 0
		return &SegmentSequenceEvaluation{
			Action:       SegmentSequenceActionNextSegment,
			SegmentDelta: delta,
			Accept:       true,
			Hold:         false,
		}
	}

	if delta < 0 {
		if IsBackwardSnapAllowed(s.state, s.config) {
			s.state.SegmentJumpCount = 0
			return &SegmentSequenceEvaluation{
				Action:       SegmentSequenceActionAccept,
				SegmentDelta: delta,
				Accept:       true,
				Hold:         false,
			}
		}
		return &SegmentSequenceEvaluation{
			Action:       SegmentSequenceActionHoldReverse,
			SegmentDelta: delta,
			Accept:       false,
			Hold:         true,
		}
	}

	s.state.SegmentJumpCount++
	if s.canRecoverSegmentJump(point, result, best) {
		s.state.SegmentJumpCount = 0
		s.state.SkippedSegmentCount++
		return &SegmentSequenceEvaluation{
			Action:       SegmentSequenceActionRecovery,
			SegmentDelta: delta,
			Accept:       true,
			Hold:         false,
		}
	}

	return &SegmentSequenceEvaluation{
		Action:       SegmentSequenceActionHoldJump,
		SegmentDelta: delta,
		Accept:       false,
		Hold:         true,
	}
}

func (s *Snapper) applyHeldSegmentJumpResult(candidateResult *SnapResult, reason, rejectedReason string) *SnapResult {
	state := s.state
	minConf := s.holdLastSegmentMinConfidence()
	conf := candidateResult.Confidence * 0.5
	if conf < minConf {
		conf = minConf
	}

	if state.LastValidSegmentOrder > 0 {
		out := *candidateResult
		out.SnappedPoint = state.LastValidSnappedPoint
		out.SegmentID = state.LastValidSegmentID
		out.SegmentOrder = state.LastValidSegmentOrder
		out.Progress = state.LastValidProgress
		out.Confidence = clampConfidence(conf)
		out.IsOffRoute = false
		out.HeldSegment = true
		out.HeldReason = reason
		out.RejectedReason = rejectedReason
		return &out
	}

	out := *candidateResult
	out.Confidence = clampConfidence(conf)
	out.IsOffRoute = false
	out.HeldSegment = true
	out.HeldReason = reason
	out.RejectedReason = rejectedReason
	return &out
}

func (s *Snapper) candidateFromLastValid(point GPSPoint) *Candidate {
	order := s.state.LastValidSegmentOrder
	if order <= 0 {
		return s.state.LastBest
	}
	seg := s.segmentByOrder(order)
	if seg == nil {
		return s.state.LastBest
	}

	segLen := seg.ToMeasure - seg.FromMeasure
	rel := 0.0
	if segLen > 0 {
		rel = s.state.LastValidProgress * segLen
	}
	proj := ProjectionCandidate{
		Point:         s.state.LastValidSnappedPoint,
		Measure:       rel,
		DistanceMeter: DistanceMeter(point.Point, s.state.LastValidSnappedPoint),
		LineIndex:     0,
	}
	candidate := s.candidateFromProjection(*seg, point, proj)
	candidate.Measure = seg.FromMeasure + rel
	return &candidate
}

func (s *Snapper) stabilizeSegmentSequence(point GPSPoint, best *Candidate, result *SnapResult, evaluation *SegmentSequenceEvaluation) (*SnapResult, *Candidate) {
	if evaluation == nil || best == nil || !s.config.SegmentSequenceValidation {
		return nil, nil
	}

	if evaluation.Accept {
		if evaluation.Action == SegmentSequenceActionRecovery {
			out := *result
			out.HeldSegment = true
			out.HeldReason = "recovered_after_consistent_segment_jump"
			out.RejectedReason = ""
			return &out, best
		}
		return nil, nil
	}

	if evaluation.Action == SegmentSequenceActionHoldJump {
		held := s.applyHeldSegmentJumpResult(result, "segment_jump_not_allowed", "skipped_segment_order")
		heldBest := s.candidateFromLastValid(point)
		return held, heldBest
	}

	if evaluation.Action == SegmentSequenceActionHoldReverse {
		held := s.applyHeldSegmentJumpResult(result, "segment_reverse_not_validated", "segment_reverse_not_validated")
		heldBest := s.candidateFromLastValid(point)
		if heldBest == nil && s.state.LastBest != nil {
			fallback := s.candidateOnSegment(s.state.LastBest.Segment, point)
			heldBest = &fallback
		}
		return held, heldBest
	}

	return nil, nil
}

func (s *Snapper) updateLastValidSegment(result *SnapResult, best *Candidate, segmentSeqEval *SegmentSequenceEvaluation) {
	if !s.config.SegmentSequenceValidation || best == nil || result == nil {
		return
	}
	if result.HeldReason == "segment_jump_not_allowed" {
		return
	}
	if segmentSeqEval != nil && !segmentSeqEval.Accept {
		return
	}
	if result.SegmentOrder <= 0 {
		return
	}

	s.state.LastValidSegmentID = result.SegmentID
	s.state.LastValidSegmentOrder = result.SegmentOrder
	s.state.LastValidProgress = result.Progress
	s.state.LastValidSnappedPoint = result.SnappedPoint
}

func RejectSegmentJumpCandidate(state *ViterbiState, c Candidate, segmentCount int, cfg Config) bool {
	if !cfg.SegmentSequenceValidation || state.LastBest == nil {
		return false
	}

	fromOrder := ResolveLastValidSegmentOrder(state)
	if fromOrder <= 0 {
		return false
	}

	delta := ComputeSegmentOrderDelta(fromOrder, c.Segment.Order, segmentCount, cfg.Looping)
	return delta > 1
}

func ApplySegmentSequenceResultMetadata(state *ViterbiState, cfg Config, result *SnapResult) *SnapResult {
	if !cfg.SegmentSequenceValidation || result == nil {
		return result
	}

	out := *result
	out.SegmentJumpCount = state.SegmentJumpCount
	out.SkippedSegmentCount = state.SkippedSegmentCount
	return &out
}
