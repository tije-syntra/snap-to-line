package snaptoline

const (
	DefaultOffRouteDistanceMeter      = 200.0
	DefaultOffRouteConsecutiveSamples = 5
)

func ResolveOffRouteDistanceMeter(cfg Config) float64 {
	if cfg.OffRouteDistanceMeter > 0 {
		return cfg.OffRouteDistanceMeter
	}
	return DefaultOffRouteDistanceMeter
}

func ResolveOffRouteConsecutiveSamples(cfg Config) int {
	if cfg.OffRouteConsecutiveSamples > 0 {
		return cfg.OffRouteConsecutiveSamples
	}
	return DefaultOffRouteConsecutiveSamples
}

func UpdateOffRouteCount(state *ViterbiState, distanceToRoute float64, cfg Config) bool {
	threshold := ResolveOffRouteDistanceMeter(cfg)
	if distanceToRoute > threshold {
		state.OffRouteCount++
	} else {
		state.OffRouteCount = 0
	}
	return state.OffRouteCount >= ResolveOffRouteConsecutiveSamples(cfg)
}

func ApplyConsecutiveOffRouteDetection(state *ViterbiState, cfg Config, result *SnapResult) *SnapResult {
	if result == nil || !cfg.OffRouteDetection {
		return result
	}

	consecutiveOffRoute := UpdateOffRouteCount(state, result.DistanceMeter, cfg)
	out := *result
	out.IsOffRoute = result.IsOffRoute || consecutiveOffRoute
	out.OffRouteCount = state.OffRouteCount
	return &out
}
