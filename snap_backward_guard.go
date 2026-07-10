package snaptoline

import "math"

const backwardMeasureEpsilonM = 0.5

func (s *Snapper) isBackwardSegmentTransition(best *Candidate) bool {
	if best == nil || s.state.LastBest == nil || !s.config.PreventBackwardTransition {
		return false
	}
	ref := s.state.LastBest
	if best.Segment.Order >= ref.Segment.Order {
		return false
	}
	return !isLoopWrapTransition(ref.Segment.Order, best.Segment.Order, len(s.segments), s.config.Looping)
}

// enforceForwardSegmentOrder rejects any candidate on a lower segment order than the last snap.
func (s *Snapper) enforceForwardSegmentOrder(best *Candidate, point GPSPoint) *Candidate {
	if s.shouldAllowBackwardSnap() {
		return best
	}
	if !s.isBackwardSegmentTransition(best) {
		return best
	}
	fallback := s.candidateOnSegment(s.state.LastBest.Segment, point)
	return &fallback
}

// enforceNoBackwardSnap prevents route regression while still allowing forward creep when GPS progresses.
func (s *Snapper) enforceNoBackwardSnap(best *Candidate, point GPSPoint) (*SnapResult, *Candidate) {
	if !s.noBackwardSnapEnabled() || s.state.LastBest == nil || best == nil {
		return nil, nil
	}

	ref := s.state.LastBest
	loopWrap := isLoopWrapTransition(ref.Segment.Order, best.Segment.Order, len(s.segments), s.config.Looping)
	if loopWrap {
		return nil, nil
	}

	backward := false
	if best.Segment.Order < ref.Segment.Order {
		backward = true
	}
	if best.Measure < ref.Measure-backwardMeasureEpsilonM {
		backward = true
	}
	if !backward {
		return nil, nil
	}

	if best.Measure < ref.Measure-backwardMeasureEpsilonM && !s.gpsMovesForwardAlongRoute(point, ref) {
		return s.freezeAtLastSnap(point, "no_backward")
	}

	return s.advanceAlongRoute(point, "no_backward")
}

func (s *Snapper) gpsMovesForwardAlongRoute(point GPSPoint, ref *Candidate) bool {
	if s.state.LastPoint == nil || ref == nil {
		return false
	}
	movement := DistanceMeter(s.state.LastPoint.Point, point.Point)
	minMov := s.config.MinMovementMeter
	if minMov <= 0 {
		minMov = 3
	}
	if movement < minMov {
		return false
	}
	moveBearing := BearingBetween(s.state.LastPoint.Point, point.Point)
	return bearingDiffDeg(moveBearing, ref.LineBearing) <= 90
}

// shouldCreepForwardFromPrevious is true when GPS progresses along the route bearing,
// or when raw GPS drifts away from the last snap while moving in a forward direction.
func (s *Snapper) shouldCreepForwardFromPrevious(point GPSPoint, ref *Candidate) bool {
	if s.gpsMovesForwardAlongRoute(point, ref) {
		return true
	}
	if s.state.LastPoint == nil || ref == nil {
		return false
	}
	rawSnap := DistanceMeter(point.Point, ref.SnappedPoint)
	prevRawSnap := DistanceMeter(s.state.LastPoint.Point, ref.SnappedPoint)
	movement := DistanceMeter(s.state.LastPoint.Point, point.Point)
	if movement < 0.5 {
		return false
	}

	maxSnap := s.config.MaxSnapDistanceMeter
	moveBearing := BearingBetween(s.state.LastPoint.Point, point.Point)
	if bearingDiffDeg(moveBearing, ref.LineBearing) > 90 {
		return false
	}
	if maxSnap > 0 && rawSnap > maxSnap {
		onSeg := s.candidateOnSegment(ref.Segment, point)
		if onSeg.DistanceMeter <= maxSnap {
			return s.gpsMovesForwardAlongRoute(point, ref)
		}
		return false
	}
	if rawSnap <= prevRawSnap+0.5 {
		return false
	}
	return true
}

func bearingDiffDeg(a, b float64) float64 {
	d := math.Mod(a-b+540, 360) - 180
	if d < 0 {
		d = -d
	}
	return d
}

// advanceAlongRoute projects onto the active segment forward-only, then creeps by GPS movement if needed.
func (s *Snapper) advanceAlongRoute(point GPSPoint, reason string) (*SnapResult, *Candidate) {
	ref := s.state.LastBest
	if ref == nil {
		return nil, nil
	}

	maxDist := s.holdLastSegmentMaxDistM()
	if maxDist < 500 {
		maxDist = 500
	}
	result, candidate := s.clampToPreviousSegmentWithMaxDist(point, maxDist)
	if candidate != nil && candidate.Measure > ref.Measure+backwardMeasureEpsilonM {
		if result != nil {
			result = s.applyStabilizedResult(result, reason)
		}
		return result, candidate
	}

	if s.gpsMovesForwardAlongRoute(point, ref) {
		if crept, c := s.creepForwardOnSegment(point, ref); crept != nil {
			crept = s.applyStabilizedResult(crept, reason)
			return crept, c
		}
	}

	return s.freezeAtLastSnap(point, reason)
}

func (s *Snapper) creepForwardOnSegment(point GPSPoint, ref *Candidate) (*SnapResult, *Candidate) {
	if ref == nil {
		return nil, nil
	}

	creep := s.plausibleRawGPSMovementM(point)
	if maxFwd := s.config.MaxForwardSnapMeter; maxFwd > 0 && creep > maxFwd {
		creep = maxFwd
	}
	advSlack := s.config.MeasureAdvanceSlackMeter
	if advSlack <= 0 {
		advSlack = DefaultRouteMeasureAdvanceSlackMeter
	}
	if s.state.LastPoint != nil {
		movement := DistanceMeter(s.state.LastPoint.Point, point.Point)
		cap := movement*1.5 + advSlack
		if creep > cap {
			creep = cap
		}
	}
	if creep < 0.5 {
		return nil, nil
	}

	targetM := ref.Measure + creep
	seg := s.segmentAtRouteMeasure(targetM)
	if seg == nil || seg.Order < ref.Segment.Order {
		seg = &ref.Segment
		targetM = math.Min(ref.Measure+creep, ref.Segment.ToMeasure)
	}

	return s.candidateAtRouteMeasure(*seg, targetM, point)
}

func (s *Snapper) noBackwardSnapEnabled() bool {
	if s.config.NoBackwardSnap {
		return true
	}
	return s.config.PreventBackwardTransition
}

func (s *Snapper) finishSnap(candidates []Candidate, best *Candidate, result *SnapResult, point GPSPoint, reverseEval *ReverseEvaluation, segmentSeqEval *SegmentSequenceEvaluation) *SnapResult {
	if best == nil {
		return result
	}
	if result == nil {
		result = s.resultFromCandidate(*best, point)
	}
	if adjusted, fBest := s.enforceNoBackwardSnap(best, point); adjusted != nil {
		best = fBest
		result = adjusted
	}
	if result != nil {
		result = ApplyConsecutiveOffRouteDetection(s.state, s.config, result)
		result = ApplyGpsJumpResultMetadata(s.state, s.config, result)
		result = ApplyReverseResultMetadata(s.state, s.config, result, reverseEval)
		result = ApplySegmentSequenceResultMetadata(s.state, s.config, result)
	}
	if segmentSeqEval == nil || segmentSeqEval.Accept {
		s.updateLastValidSegment(result, best, segmentSeqEval)
	}
	s.updateBranchLock(best, point)
	s.annotateBranchLock(result)
	s.annotateStopContext(result, point)
	s.commitSnapState(candidates, best, point, result)
	return result
}
