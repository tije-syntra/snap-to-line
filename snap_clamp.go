package snaptoline

func (s *Snapper) shouldClampBackward(result *SnapResult, point GPSPoint) bool {
	if !s.config.PreventBackwardTransition || s.state.LastBest == nil {
		return false
	}

	prevOrder := s.state.LastBest.Segment.Order
	if result.SegmentOrder >= prevOrder {
		return false
	}
	if isLoopWrapTransition(prevOrder, result.SegmentOrder, len(s.segments), s.config.Looping) {
		return false
	}

	minConf := s.config.ClampBackwardMinConfidence
	if minConf <= 0 {
		return false
	}

	dwell := s.config.ClampDwellSpeedKmh
	if dwell <= 0 {
		dwell = 8
	}

	tol := s.config.MeasureRegressionToleranceMeter
	if tol <= 0 {
		tol = 30
	}

	lowConf := result.Confidence < minConf
	slow := point.Speed <= dwell
	measureBack := s.RouteMeasure(result.SegmentOrder, result.Progress) <
		s.state.LastBest.Measure-tol

	return lowConf || slow || measureBack
}

func (s *Snapper) clampToPreviousSegment(point GPSPoint) *SnapResult {
	if s.state.LastBest == nil {
		return nil
	}
	return SnapResultFromSegment(s.state.LastBest.Segment, s.stops, point, s.state.LastPoint, s.config)
}

func (s *Snapper) candidateOnSegment(seg Segment, point GPSPoint) Candidate {
	proj := ProjectPointOnLine(seg.Geometry, point.Point)
	absMeasure := seg.FromMeasure + proj.Measure
	lineBearing := BearingAtMeasure(seg.Geometry, proj.Measure)

	busBearing, hasBearing := resolveBusBearing(point, s.state.LastPoint, s.config)
	weaken := shouldWeakenDirectionValidation(point, s.state.LastPoint, s.config)
	emission := EmissionScore(proj.DistanceMeter, s.config.MaxSnapDistanceMeter)
	dirScore, _ := scoreDirection(busBearing, hasBearing, lineBearing, s.config, weaken)
	tripScore := TripDirectionScore(seg.Direction, s.config.TripDirection)

	return Candidate{
		Segment:            seg,
		Measure:            absMeasure,
		SnappedPoint:       proj.Point,
		DistanceMeter:      proj.DistanceMeter,
		LineBearing:        lineBearing,
		EmissionScore:      emission,
		DirectionScore:     dirScore,
		TripDirectionScore: tripScore,
	}
}

func (s *Snapper) resultFromCandidate(best Candidate, point GPSPoint) *SnapResult {
	busBearing, hasBearing := resolveBusBearing(point, s.state.LastPoint, s.config)
	weaken := shouldWeakenDirectionValidation(point, s.state.LastPoint, s.config)
	_, directionDiff := scoreDirection(busBearing, hasBearing, best.LineBearing, s.config, weaken)

	if !hasBearing && s.state.LastBest != nil {
		busBearing = s.state.LastBest.LineBearing
	}

	return &SnapResult{
		OriginalPoint: point.Point,
		SnappedPoint:  best.SnappedPoint,
		SegmentID:     best.Segment.ID,
		SegmentOrder:  best.Segment.Order,
		Direction:     best.Segment.Direction,
		NearestStopID: nearestStopID(s.stops, best.SnappedPoint),
		DistanceMeter: best.DistanceMeter,
		Progress:      segmentProgress(best.Segment, best.Measure),
		BusBearing:    busBearing,
		LineBearing:   best.LineBearing,
		DirectionDiff: directionDiff,
		Confidence:    clampConfidence(confidenceFromScores(best)),
		IsOffRoute:    best.DistanceMeter > s.config.MaxSnapDistanceMeter,
	}
}
