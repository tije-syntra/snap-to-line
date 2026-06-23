package snaptoline

// capForwardSnapAdvance limits how far ahead on the route line a snap may move per tick.
func (s *Snapper) capForwardSnapAdvance(best *Candidate, point GPSPoint) (*SnapResult, *Candidate) {
	maxFwd := s.config.MaxForwardSnapMeter
	if maxFwd <= 0 || s.state.LastBest == nil || best == nil {
		return nil, nil
	}

	ref := s.state.LastBest
	loopWrap := isLoopWrapTransition(ref.Segment.Order, best.Segment.Order, len(s.segments), s.config.Looping)
	if loopWrap {
		return nil, nil
	}

	lastM := ref.Measure
	newM := best.Measure
	if newM <= lastM+maxFwd {
		return nil, nil
	}

	targetM := lastM + maxFwd
	seg := s.segmentAtRouteMeasure(targetM)
	if seg == nil {
		return nil, nil
	}

	result, candidate := s.candidateAtRouteMeasure(*seg, targetM, point)
	if result == nil {
		return nil, nil
	}
	minConf := s.holdLastSegmentMinConfidence()
	if result.Confidence < minConf {
		result.Confidence = minConf
	}
	result.HeldSegment = true
	result.HeldReason = "forward_cap"
	result.IsOffRoute = false
	return result, candidate
}

func (s *Snapper) segmentAtRouteMeasure(m float64) *Segment {
	for i := range s.segments {
		seg := &s.segments[i]
		if m >= seg.FromMeasure && m <= seg.ToMeasure {
			return seg
		}
	}
	if len(s.segments) == 0 {
		return nil
	}
	last := &s.segments[len(s.segments)-1]
	if m > last.ToMeasure {
		return last
	}
	first := &s.segments[0]
	if m < first.FromMeasure {
		return first
	}
	return nil
}

func (s *Snapper) candidateAtRouteMeasure(seg Segment, absMeasure float64, point GPSPoint) (*SnapResult, *Candidate) {
	rel := absMeasure - seg.FromMeasure
	if rel < 0 {
		rel = 0
	}
	segLen := seg.ToMeasure - seg.FromMeasure
	if segLen > 0 && rel > segLen {
		rel = segLen
	}
	pt, _ := PointAtMeasure(seg.Geometry, rel)
	proj := ProjectionCandidate{
		Point:         pt,
		Measure:       rel,
		DistanceMeter: DistanceMeter(point.Point, pt),
	}
	candidate := s.candidateFromProjection(seg, point, proj)
	candidate.Measure = seg.FromMeasure + rel
	result := s.resultFromCandidate(candidate, point)
	return result, &candidate
}
