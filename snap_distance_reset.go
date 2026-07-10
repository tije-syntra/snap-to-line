package snaptoline

import "github.com/paulmach/orb"

func snapDistanceResetEnabled(cfg Config) bool {
	return cfg.SnapDistanceResetOnGrow
}

func snapDistanceGrowResetTicks(cfg Config) int {
	if cfg.SnapDistanceGrowResetTicks > 0 {
		return cfg.SnapDistanceGrowResetTicks
	}
	return DefaultSnapDistanceGrowResetTicks
}

func snapDistanceGrowMinDeltaM(cfg Config) float64 {
	if cfg.SnapDistanceGrowMinDeltaM > 0 {
		return cfg.SnapDistanceGrowMinDeltaM
	}
	return DefaultSnapDistanceGrowMinDeltaM
}

func snapDistanceResetMinMeter(cfg Config) float64 {
	if cfg.SnapDistanceResetMinMeter > 0 {
		return cfg.SnapDistanceResetMinMeter
	}
	maxSnap := cfg.MaxSnapDistanceMeter
	if maxSnap > 0 {
		return maxSnap + 4
	}
	return DefaultSnapDistanceResetMinMeter
}

func snapDistanceResetMaxMeter(cfg Config) float64 {
	if cfg.SnapDistanceResetMaxMeter == 0 {
		return 0
	}
	if cfg.SnapDistanceResetMaxMeter > 0 {
		return cfg.SnapDistanceResetMaxMeter
	}
	return DefaultSnapDistanceResetMaxMeter
}

// maybeResetSnapDistance clears Viterbi tracking when raw-to-snap distance keeps growing,
// so the next tick re-projects from GPS instead of freezing at a stale position.
func (s *Snapper) maybeResetSnapDistance(point GPSPoint) bool {
	if s.state.LastBest == nil {
		return false
	}

	ref := s.state.LastBest
	lateralDist := s.lateralDistFromLastSnapM(point)
	if maxReset := snapDistanceResetMaxMeter(s.config); maxReset > 0 && lateralDist >= maxReset {
		s.resetSnapTrackingState()
		return true
	}

	if !snapDistanceResetEnabled(s.config) {
		return false
	}

	dist := DistanceMeter(point.Point, ref.SnappedPoint)
	minDist := snapDistanceResetMinMeter(s.config)
	if dist < minDist {
		s.state.GrowingSnapDistTicks = 0
		return false
	}

	maxSnap := s.config.MaxSnapDistanceMeter
	if maxSnap <= 0 {
		maxSnap = 28
	}
	if s.state.LastPoint != nil && dist < maxSnap*2 {
		moveBearing := BearingBetween(s.state.LastPoint.Point, point.Point)
		if bearingDiffDeg(moveBearing, ref.LineBearing) > 90 {
			s.state.GrowingSnapDistTicks = 0
			return false
		}
	}

	prevDist := s.state.LastOutputSnapDistanceM
	growDelta := snapDistanceGrowMinDeltaM(s.config)
	if prevDist > 0 && dist > prevDist+growDelta {
		s.state.GrowingSnapDistTicks++
	} else if dist <= prevDist {
		s.state.GrowingSnapDistTicks = 0
	}

	if s.state.GrowingSnapDistTicks < snapDistanceGrowResetTicks(s.config) {
		return false
	}

	s.resetSnapTrackingState()
	return true
}

func (s *Snapper) resetSnapTrackingState() {
	activeDir := s.state.ActiveDirection
	preserved := struct {
		lastValidSegmentID        string
		lastValidSegmentOrder     int
		lastValidProgress         float64
		lastValidSnappedPoint     orb.Point
		segmentJumpCount          int
		skippedSegmentCount       int
	}{
		lastValidSegmentID:    s.state.LastValidSegmentID,
		lastValidSegmentOrder: s.state.LastValidSegmentOrder,
		lastValidProgress:     s.state.LastValidProgress,
		lastValidSnappedPoint: s.state.LastValidSnappedPoint,
		segmentJumpCount:      s.state.SegmentJumpCount,
		skippedSegmentCount:   s.state.SkippedSegmentCount,
	}
	s.state.Reset()
	s.state.ActiveDirection = activeDir
	s.state.GrowingSnapDistTicks = 0
	s.state.LastOutputSnapDistanceM = 0
	s.state.LastValidSegmentID = preserved.lastValidSegmentID
	s.state.LastValidSegmentOrder = preserved.lastValidSegmentOrder
	s.state.LastValidProgress = preserved.lastValidProgress
	s.state.LastValidSnappedPoint = preserved.lastValidSnappedPoint
	s.state.SegmentJumpCount = preserved.segmentJumpCount
	s.state.SkippedSegmentCount = preserved.skippedSegmentCount
}

func markDistanceReset(result *SnapResult, distReset bool) *SnapResult {
	if distReset && result != nil && result.HeldReason == "" {
		result.HeldReason = "snap_distance_reset"
	}
	return result
}

func (s *Snapper) lateralDistFromLastSnapM(point GPSPoint) float64 {
	if s.state.LastBest == nil {
		return 0
	}
	proj := projectOntoSegment(s.state.LastBest.Segment, point, s.state.LastPoint, s.state, s.config)
	return proj.DistanceMeter
}
