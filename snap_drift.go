package snaptoline

import "math"

const gpsDriftResnapMinM = 20.0

func (s *Snapper) rawGPSDriftedOffSnap(point GPSPoint) bool {
	ref := s.state.LastBest
	if ref == nil {
		return false
	}
	maxSnap := s.config.MaxSnapDistanceMeter
	if maxSnap <= 0 {
		maxSnap = 28
	}
	dist := DistanceMeter(point.Point, ref.SnappedPoint)
	minDrift := math.Max(maxSnap*1.15, gpsDriftResnapMinM)
	return dist >= minDrift
}

// resnapOnActiveSegmentWhenDrifted re-projects onto the active segment when raw GPS has
// drifted far from the last snap but may still be near the route polyline.
func (s *Snapper) resnapOnActiveSegmentWhenDrifted(best *Candidate, point GPSPoint) *Candidate {
	if s.state.LastBest == nil || !s.rawGPSDriftedOffSnap(point) {
		return best
	}
	if s.isWildGPSJump(point) {
		return best
	}
	ref := s.state.LastBest

	maxDist := s.holdLastSegmentMaxDistM()
	maxSnap := s.config.MaxSnapDistanceMeter
	if maxSnap <= 0 {
		maxSnap = 28
	}
	if maxDist < maxSnap*2 {
		maxDist = maxSnap * 2
	}

	_, candidate := s.clampToPreviousSegmentWithMaxDist(point, maxDist)
	if candidate == nil {
		return best
	}

	tol := s.config.MeasureRegressionToleranceMeter
	if tol <= 0 {
		tol = DefaultRouteMeasureRegressionToleranceMeter
	}
	if candidate.Measure < ref.Measure-tol {
		return best
	}

	gpsMovement := 0.0
	if s.state.LastPoint != nil {
		gpsMovement = DistanceMeter(s.state.LastPoint.Point, point.Point)
	}
	maxAdv := maxAllowedMeasureAdvanceM(s.config, gpsMovement)
	if candidate.Measure > ref.Measure+maxAdv {
		return best
	}

	if best != nil {
		bestRaw := DistanceMeter(point.Point, best.SnappedPoint)
		candRaw := DistanceMeter(point.Point, candidate.SnappedPoint)
		if candRaw >= bestRaw {
			return best
		}
	}
	return candidate
}

// pickNearestForwardCandidate picks the closest candidate on the same or next segment order.
func pickNearestForwardCandidate(candidates []Candidate, state *ViterbiState, segmentCount int, looping bool) *Candidate {
	if len(candidates) == 0 {
		return nil
	}
	refOrder := 0
	if state != nil && state.LastBest != nil {
		refOrder = state.LastBest.Segment.Order
	}
	maxOrder := refOrder + 1
	if refOrder <= 0 {
		maxOrder = 0
	}

	var best *Candidate
	for i := range candidates {
		c := candidates[i]
		if refOrder > 0 {
			if c.Segment.Order < refOrder &&
				!isLoopWrapTransition(refOrder, c.Segment.Order, segmentCount, looping) {
				continue
			}
			if maxOrder > 0 && c.Segment.Order > maxOrder &&
				!isLoopWrapTransition(refOrder, c.Segment.Order, segmentCount, looping) {
				continue
			}
		}
		if best == nil || c.DistanceMeter < best.DistanceMeter {
			copy := c
			best = &copy
		}
	}
	return best
}
