package snaptoline

// stabilizeWildGPSJump limits snap movement when raw GPS jumps implausibly far.
// Backward / measure regression freezes the last snap; forward advance is capped by MaxForwardSnapMeter.
func (s *Snapper) stabilizeWildGPSJump(best *Candidate, point GPSPoint) (*SnapResult, *Candidate) {
	if !s.wildGPSStabilizeEnabled() || s.state.LastBest == nil || best == nil || s.state.LastPoint == nil {
		return nil, nil
	}
	if !s.isWildGPSJump(point) {
		return nil, nil
	}

	ref := s.state.LastBest
	tol := s.config.MeasureRegressionToleranceMeter
	if tol <= 0 {
		tol = DefaultRouteMeasureRegressionToleranceMeter
	}

	loopWrap := isLoopWrapTransition(ref.Segment.Order, best.Segment.Order, len(s.segments), s.config.Looping)
	lastM := ref.Measure
	newM := best.Measure

	if best.Segment.Order < ref.Segment.Order && !loopWrap {
		return s.advanceAlongRoute(point, "wild_gps_backward")
	}

	if newM < lastM-tol && !loopWrap {
		if s.gpsMovesForwardAlongRoute(point, ref) {
			return s.advanceAlongRoute(point, "wild_gps_regression")
		}
		return s.freezeAtLastSnap(point, "wild_gps_regression")
	}

	return nil, nil
}

func (s *Snapper) wildGPSStabilizeEnabled() bool {
	return s.config.WildGPSStabilize
}

func (s *Snapper) rawGPSMovementM(point GPSPoint) float64 {
	if s.state.LastPoint == nil {
		return 0
	}
	return DistanceMeter(s.state.LastPoint.Point, point.Point)
}

func (s *Snapper) isWildGPSJump(point GPSPoint) bool {
	raw := s.rawGPSMovementM(point)
	minM := s.config.WildGPSJumpMinMeter
	if minM <= 0 {
		minM = DefaultRouteWildGPSJumpMinMeter
	}
	if raw < minM {
		return false
	}

	mult := s.config.WildGPSJumpMultiplier
	if mult <= 0 {
		mult = DefaultRouteWildGPSJumpMultiplier
	}
	return raw > s.plausibleRawGPSMovementM(point)*mult
}

func (s *Snapper) plausibleRawGPSMovementM(point GPSPoint) float64 {
	slack := s.config.MeasureAdvanceSlackMeter
	if slack <= 0 {
		slack = DefaultRouteMeasureAdvanceSlackMeter
	}

	deltaSec := 1.0
	if point.Timestamp > 0 && s.state.LastTimestamp > 0 {
		deltaSec = float64(point.Timestamp-s.state.LastTimestamp) / 1000.0
	}
	if deltaSec < 0.3 {
		deltaSec = 0.3
	}
	if deltaSec > 30 {
		deltaSec = 30
	}

	speedKmh := point.Speed
	if speedKmh <= 0 && s.state.LastPoint != nil {
		speedKmh = s.state.LastPoint.Speed
	}
	if speedKmh <= 0 {
		speedKmh = 40
	}

	return (speedKmh/3.6)*deltaSec + slack
}

func (s *Snapper) freezeAtLastSnap(point GPSPoint, reason string) (*SnapResult, *Candidate) {
	return s.freezeAtRefCandidate(point, s.state.LastBest, reason)
}

func (s *Snapper) freezeAtRefCandidate(point GPSPoint, ref *Candidate, reason string) (*SnapResult, *Candidate) {
	if ref == nil {
		return nil, nil
	}
	seg := ref.Segment
	prevRel := ref.Measure - seg.FromMeasure
	proj := ProjectionCandidate{
		Point:         ref.SnappedPoint,
		Measure:       prevRel,
		DistanceMeter: DistanceMeter(point.Point, ref.SnappedPoint),
	}
	candidate := s.candidateFromProjection(seg, point, proj)
	candidate.Measure = ref.Measure
	result := s.resultFromCandidate(candidate, point)
	result = s.applyStabilizedResult(result, reason)
	return result, &candidate
}

func (s *Snapper) snapAtRouteMeasure(seg Segment, absMeasure float64, point GPSPoint, reason string) (*SnapResult, *Candidate) {
	result, candidate := s.candidateAtRouteMeasure(seg, absMeasure, point)
	if result == nil {
		return nil, nil
	}
	result = s.applyStabilizedResult(result, reason)
	return result, candidate
}

func (s *Snapper) applyStabilizedResult(result *SnapResult, reason string) *SnapResult {
	minConf := s.holdLastSegmentMinConfidence()
	conf := result.Confidence * 0.6
	if conf < minConf {
		conf = minConf
	}
	result.Confidence = clampConfidence(conf)
	result.HeldSegment = true
	result.HeldReason = reason
	result.IsOffRoute = false
	result.RejectedReason = ""
	return result
}
