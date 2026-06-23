package snaptoline

func nextStopPassTolerance(cfg Config) float64 {
	tol := cfg.NextStopPassToleranceMeter
	if tol <= 0 {
		tol = DefaultRouteNextStopPassToleranceMeter
	}
	return tol
}

func hasPassedSegmentDestination(measure float64, seg Segment, cfg Config) bool {
	return measure >= seg.ToMeasure-nextStopPassTolerance(cfg)
}

func departingSegmentMeasure(state *ViterbiState, seg Segment, point GPSPoint) float64 {
	measure := seg.FromMeasure
	if state != nil && state.LastBest != nil && state.LastBest.Segment.Order == seg.Order {
		if state.LastBest.Measure > measure {
			measure = state.LastBest.Measure
		}
	}
	proj := ProjectPointOnLine(seg.Geometry, point.Point)
	candidate := seg.FromMeasure + proj.Measure
	if candidate > measure {
		return candidate
	}
	return measure
}

func rejectPrematureSegmentSwitch(
	state *ViterbiState,
	c Candidate,
	segmentCount int,
	point GPSPoint,
	cfg Config,
) bool {
	if !cfg.RequireNextStopBeforeSegmentSwitch || state == nil || state.LastBest == nil {
		return false
	}

	from := state.LastBest.Segment
	to := c.Segment
	if from.ID == to.ID {
		return false
	}

	looping := cfg.Looping
	fromOrder := from.Order
	toOrder := to.Order
	loopWrap := isLoopWrapTransition(fromOrder, toOrder, segmentCount, looping)

	if toOrder < fromOrder && !loopWrap {
		return false // handled by backward guards
	}

	departMeasure := departingSegmentMeasure(state, from, point)
	if !hasPassedSegmentDestination(departMeasure, from, cfg) {
		return true
	}

	if loopWrap {
		return false
	}

	if toOrder > fromOrder+1 {
		return true
	}

	return false
}

func (s *Snapper) enforceNextStopBeforeSegmentSwitch(best *Candidate, point GPSPoint) *Candidate {
	if best == nil || s.state.LastBest == nil || !s.config.RequireNextStopBeforeSegmentSwitch {
		return nil
	}
	if rejectPrematureSegmentSwitch(s.state, *best, len(s.segments), point, s.config) {
		fallback := s.candidateOnSegment(s.state.LastBest.Segment, point)
		return &fallback
	}
	return nil
}
