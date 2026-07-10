package snaptoline

import "time"

const (
	DefaultRouteTeleportDistanceMeter    = 300.0
	DefaultRouteTeleportTimeSec          = 5.0
	DefaultRouteTeleportSpeedMatchFactor = 0.8
)

func resolveTimestamp(point GPSPoint, fallback int64) int64 {
	if point.Timestamp > 0 {
		return point.Timestamp
	}
	return fallback
}

// isGpsTeleport reports whether GPS moved >= TeleportDistanceMeter within TeleportTimeSec
// without reported speed matching the implied movement.
func IsGpsTeleport(state *ViterbiState, cfg Config, point GPSPoint) bool {
	if !cfg.TeleportDetection || state.LastPoint == nil || state.LastTimestamp <= 0 {
		return false
	}

	gpsDistance := DistanceMeter(state.LastPoint.Point, point.Point)
	if gpsDistance < cfg.TeleportDistanceMeter {
		return false
	}

	nowTs := resolveTimestamp(point, time.Now().UnixMilli())
	deltaSec := float64(nowTs-state.LastTimestamp) / 1000.0
	if deltaSec <= 0 || deltaSec > cfg.TeleportTimeSec {
		return false
	}

	requiredSpeedKmh := (gpsDistance / deltaSec) * 3.6
	reportedSpeedKmh := point.Speed
	if reportedSpeedKmh <= 0 {
		reportedSpeedKmh = state.LastPoint.Speed
	}
	if reportedSpeedKmh <= 0 {
		return true
	}

	factor := cfg.TeleportSpeedMatchFactor
	if factor <= 0 {
		factor = DefaultRouteTeleportSpeedMatchFactor
	}
	return reportedSpeedKmh < requiredSpeedKmh*factor
}

func (s *Snapper) isGpsTeleport(point GPSPoint) bool {
	return IsGpsTeleport(s.state, s.config, point)
}

func (s *Snapper) stabilizeGpsTeleport(point GPSPoint) (*SnapResult, *Candidate) {
	if !s.isGpsTeleport(point) {
		return nil, nil
	}
	ref := s.state.LastBest
	if ref == nil {
		ref = s.state.LastGood
	}
	if ref == nil {
		return nil, nil
	}
	return s.freezeAtRefCandidate(point, ref, "teleport_detected")
}
