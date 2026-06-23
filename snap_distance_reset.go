package snaptoline

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

// maybeResetSnapDistance clears Viterbi tracking when raw-to-snap distance keeps growing,
// so the next tick re-projects from GPS instead of freezing at a stale position.
func (s *Snapper) maybeResetSnapDistance(point GPSPoint) bool {
	if !snapDistanceResetEnabled(s.config) || s.state.LastBest == nil {
		return false
	}

	ref := s.state.LastBest
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
	s.state.Reset()
	s.state.ActiveDirection = activeDir
	s.state.GrowingSnapDistTicks = 0
	s.state.LastOutputSnapDistanceM = 0
}
