package snaptoline

const (
	DefaultOffRouteSoftDistFraction = 0.54
	DefaultOffRouteMinConfidence    = 0.2
	DefaultOffRouteSoftConfidence   = 0.5
)

// OffRoutePolicy holds post-snap thresholds for map/UI off-route and ETA freeze.
type OffRoutePolicy struct {
	MaxSnapDistanceMeter     float64
	OffRouteSoftDistFraction float64
	OffRouteMinConfidence    float64
	OffRouteSoftConfidence   float64
}

// DefaultOffRoutePolicy returns dashboard-aligned off-route defaults (max snap 28 m).
func DefaultOffRoutePolicy() OffRoutePolicy {
	return OffRoutePolicy{
		MaxSnapDistanceMeter:     28,
		OffRouteSoftDistFraction: DefaultOffRouteSoftDistFraction,
		OffRouteMinConfidence:    DefaultOffRouteMinConfidence,
		OffRouteSoftConfidence:   DefaultOffRouteSoftConfidence,
	}
}

func (p OffRoutePolicy) Normalized() OffRoutePolicy {
	out := p
	if out.MaxSnapDistanceMeter <= 0 {
		out.MaxSnapDistanceMeter = DefaultOffRoutePolicy().MaxSnapDistanceMeter
	}
	if out.OffRouteSoftDistFraction <= 0 {
		out.OffRouteSoftDistFraction = DefaultOffRouteSoftDistFraction
	}
	if out.OffRouteMinConfidence <= 0 {
		out.OffRouteMinConfidence = DefaultOffRouteMinConfidence
	}
	if out.OffRouteSoftConfidence <= 0 {
		out.OffRouteSoftConfidence = DefaultOffRouteSoftConfidence
	}
	return out
}

func (p OffRoutePolicy) offRouteSoftDistM() float64 {
	p = p.Normalized()
	return p.MaxSnapDistanceMeter * p.OffRouteSoftDistFraction
}

// SnapDegraded reports unusable snap output (missing segment, rejected, held+off-route).
func SnapDegraded(result *SnapResult) bool {
	if result == nil {
		return true
	}
	if result.SegmentID == "" || result.SegmentOrder <= 0 {
		return true
	}
	if result.RejectedReason != "" {
		return true
	}
	return result.HeldSegment && result.IsOffRoute
}

// MapOffRoute decides map/UI off-route from snap result and policy.
// Held-segment masking applies: held snaps are not off-route on the map.
func MapOffRoute(result *SnapResult, snapDegraded bool, policy OffRoutePolicy) bool {
	policy = policy.Normalized()
	if result == nil || snapDegraded {
		return true
	}
	isOffRoute := result.IsOffRoute
	if result.HeldSegment {
		isOffRoute = false
	}
	if isOffRoute || result.HeldSegment {
		return isOffRoute
	}
	if result.Confidence < policy.OffRouteMinConfidence {
		return true
	}
	soft := policy.offRouteSoftDistM()
	if result.DistanceMeter > soft && result.Confidence < policy.OffRouteSoftConfidence {
		return true
	}
	return false
}

// EtaSnapUnreliable reports when snap quality is too poor to refresh ETA / PIS.
// Unlike MapOffRoute, held-segment masking is ignored.
func EtaSnapUnreliable(result *SnapResult, snapDegraded bool, policy OffRoutePolicy) bool {
	policy = policy.Normalized()
	if result == nil || snapDegraded {
		return true
	}
	if result.IsOffRoute {
		return true
	}
	if result.DistanceMeter > policy.MaxSnapDistanceMeter {
		return true
	}
	soft := policy.offRouteSoftDistM()
	if result.DistanceMeter > soft && result.Confidence < policy.OffRouteSoftConfidence {
		return true
	}
	if result.Confidence < policy.OffRouteMinConfidence {
		return true
	}
	return false
}

// EtaSnapReliableForPublish is true when ETA/PIS may be refreshed from this snap.
func EtaSnapReliableForPublish(result *SnapResult, snapDegraded bool, policy OffRoutePolicy) bool {
	if EtaSnapUnreliable(result, snapDegraded, policy) {
		return false
	}
	return result.SegmentOrder > 0
}
