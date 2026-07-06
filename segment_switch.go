package snaptoline

import "github.com/paulmach/orb"

func nextStopPassTolerance(cfg Config) float64 {
	tol := cfg.NextStopPassToleranceMeter
	if tol <= 0 {
		tol = DefaultRouteNextStopPassToleranceMeter
	}
	return tol
}

func segmentSwitchStopRadius(cfg Config) float64 {
	r := cfg.SegmentSwitchStopRadiusMeter
	if r <= 0 {
		r = DefaultRouteSegmentSwitchStopRadiusMeter
	}
	return r
}

func isSegmentOrderChange(state *ViterbiState, c Candidate) bool {
	if state == nil || state.LastBest == nil {
		return false
	}
	return state.LastBest.Segment.ID != c.Segment.ID
}

// segmentSwitchGateStop returns the halte where two consecutive segments meet.
func segmentSwitchGateStop(from, to Segment, segmentCount int, looping bool) (orb.Point, bool) {
	_ = segmentCount
	_ = looping
	if from.ToStop.ID != "" {
		return from.ToStop.Point, true
	}
	if to.FromStop.ID != "" {
		return to.FromStop.Point, true
	}
	return orb.Point{}, false
}

func segmentWithOrder(segments []Segment, order int) (Segment, bool) {
	for _, seg := range segments {
		if seg.Order == order {
			return seg, true
		}
	}
	return Segment{}, false
}

func forwardDepartLatchActive(state *ViterbiState, fromOrder int) bool {
	if state == nil {
		return false
	}
	latch := state.SegmentDepart
	return latch.HasDeparted && latch.GateSegmentOrder == fromOrder
}

func updateSegmentDepartLatch(state *ViterbiState, point GPSPoint, cfg Config) {
	if state == nil || state.LastBest == nil || !cfg.RequireStopRadiusForSegmentSwitch {
		return
	}

	from := state.LastBest.Segment
	if from.Order <= 0 {
		return
	}

	latch := state.SegmentDepart
	if latch.GateSegmentOrder != from.Order {
		latch = SegmentDepartLatch{GateSegmentOrder: from.Order}
	}

	gate := from.ToStop.Point
	if from.ToStop.ID == "" {
		state.SegmentDepart = latch
		return
	}

	radius := segmentSwitchStopRadius(cfg)
	dist := DistanceMeter(point.Point, gate)
	if dist <= radius {
		latch.WasInsideRadius = true
		latch.HasDeparted = false
	} else if latch.WasInsideRadius {
		departMeasure := departingSegmentMeasure(state, from, point)
		if hasPassedSegmentDestination(departMeasure, from, cfg) {
			latch.HasDeparted = true
		}
	}

	state.SegmentDepart = latch
}

func rejectSegmentSwitchOutsideStopRadius(
	state *ViterbiState,
	c Candidate,
	segmentCount int,
	point GPSPoint,
	cfg Config,
) bool {
	if !cfg.RequireStopRadiusForSegmentSwitch || !isSegmentOrderChange(state, c) {
		return false
	}

	from := state.LastBest.Segment
	to := c.Segment
	looping := cfg.Looping
	fromOrder := from.Order
	toOrder := to.Order
	loopWrap := isLoopWrapTransition(fromOrder, toOrder, segmentCount, looping)

	if toOrder < fromOrder && !loopWrap {
		return false // backward guards handle this
	}
	if toOrder > fromOrder+1 && !loopWrap {
		return true
	}

	gate, ok := segmentSwitchGateStop(from, to, segmentCount, looping)
	if !ok {
		return true
	}

	radius := segmentSwitchStopRadius(cfg)
	if DistanceMeter(point.Point, gate) <= radius {
		return false
	}

	// Outside radius: allow forward-only latch after departing the gate halte.
	if toOrder == fromOrder+1 && !loopWrap && forwardDepartLatchActive(state, fromOrder) {
		return false
	}

	return true
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

func rejectSegmentSwitch(
	state *ViterbiState,
	c Candidate,
	segmentCount int,
	point GPSPoint,
	cfg Config,
) bool {
	if !isSegmentOrderChange(state, c) {
		return false
	}
	if cfg.RequireStopRadiusForSegmentSwitch && rejectSegmentSwitchOutsideStopRadius(state, c, segmentCount, point, cfg) {
		return true
	}
	if cfg.RequireNextStopBeforeSegmentSwitch && rejectPrematureSegmentSwitch(state, c, segmentCount, point, cfg) {
		return true
	}
	return false
}

func (s *Snapper) enforceNextStopBeforeSegmentSwitch(best *Candidate, point GPSPoint) *Candidate {
	if best == nil || s.state.LastBest == nil {
		return nil
	}
	if !s.config.RequireNextStopBeforeSegmentSwitch && !s.config.RequireStopRadiusForSegmentSwitch {
		return nil
	}
	if rejectSegmentSwitch(s.state, *best, len(s.segments), point, s.config) {
		fallback := s.candidateOnSegment(s.state.LastBest.Segment, point)
		return &fallback
	}
	return nil
}

// enforceDepartLatch promotes to the next segment after the bus has departed the gate
// halte radius, even when GPS is off-route and Viterbi would hold the old segment.
func (s *Snapper) enforceDepartLatch(best *Candidate, point GPSPoint) *Candidate {
	if !s.config.RequireStopRadiusForSegmentSwitch || s.state.LastBest == nil {
		return nil
	}

	fromOrder := s.state.LastBest.Segment.Order
	if !forwardDepartLatchActive(s.state, fromOrder) {
		return nil
	}

	nextOrder := fromOrder + 1
	if best != nil && best.Segment.Order >= nextOrder {
		return nil
	}

	nextSeg, ok := segmentWithOrder(s.segments, nextOrder)
	if !ok {
		return nil
	}

	promoted := s.candidateOnSegment(nextSeg, point)
	return &promoted
}
