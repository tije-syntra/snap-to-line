package snaptoline

// Production-tuned defaults used by snap-to-line-dashboard for live MQTT GPS.
// Use LiveBusSnapConfig(stops) instead of RouteSnapConfig when you want these values.
const (
	RecommendedMaxSnapDistanceMeter            = 28
	RecommendedMaxBearingDiffDegree            = 40
	RecommendedMinMovementMeter                = 3
	RecommendedMeasureRegressionToleranceMeter = 10
	RecommendedClampBackwardMinConfidence      = 0.78
	RecommendedSegmentSwitchHysteresisLog      = 2.5
	RecommendedMeasureAdvanceSlackMeter        = 8
	RecommendedClampDwellSpeedKmh              = 10
	RecommendedSnappedJumpSlackMeter           = 2
	RecommendedSegmentSwitchStopRadiusMeter    = 20
	RecommendedLoopClosureToleranceMeter       = 15
	RecommendedLoopNextStopPassToleranceMeter  = 18
	RecommendedSnapDistanceResetMaxMeter       = 100
	RecommendedSnapDistanceGrowResetTicks      = DefaultSnapDistanceGrowResetTicks
	RecommendedSnapDistanceGrowMinDeltaM       = DefaultSnapDistanceGrowMinDeltaM
	RecommendedSnapDistanceResetMinMeter       = DefaultSnapDistanceResetMinMeter
)

// IsLoopRoute reports whether first and last stop are the same within RecommendedLoopClosureToleranceMeter.
func IsLoopRoute(stops []Stop) bool {
	return detectLoopFromStops(stops, true, RecommendedLoopClosureToleranceMeter)
}

// LiveBusSnapConfig returns the recommended snap Config for live bus tracking on gate-to-gate routes.
// It enables backward guards, segment-switch gates, folded-geometry stabilizers, and snap-distance reset.
// Loop routes disable grow-reset and use a wider next-stop pass tolerance.
func LiveBusSnapConfig(stops []Stop) Config {
	opts := []RouteSnapOption{
		WithMeasureRegressionTolerance(RecommendedMeasureRegressionToleranceMeter),
		WithClampBackwardMinConfidence(RecommendedClampBackwardMinConfidence),
		WithSegmentSwitchHysteresisLog(RecommendedSegmentSwitchHysteresisLog),
		WithMeasureAdvanceSlack(RecommendedMeasureAdvanceSlackMeter),
		WithClampDwellSpeedKmh(RecommendedClampDwellSpeedKmh),
		WithSnappedJumpSlack(RecommendedSnappedJumpSlackMeter),
		WithRequireStopRadiusForSegmentSwitch(true),
		WithSegmentSwitchStopRadiusMeter(RecommendedSegmentSwitchStopRadiusMeter),
		WithLoopClosureTolerance(RecommendedLoopClosureToleranceMeter),
		WithSnapDistanceResetMaxMeter(RecommendedSnapDistanceResetMaxMeter),
	}
	if IsLoopRoute(stops) {
		opts = append(opts,
			WithSnapDistanceResetOnGrow(false),
			WithNextStopPassToleranceMeter(RecommendedLoopNextStopPassToleranceMeter),
		)
	} else {
		opts = append(opts,
			WithSnapDistanceResetOnGrow(true),
			WithSnapDistanceGrowResetTicks(RecommendedSnapDistanceGrowResetTicks),
			WithSnapDistanceGrowMinDeltaM(RecommendedSnapDistanceGrowMinDeltaM),
			WithSnapDistanceResetMinMeter(RecommendedSnapDistanceResetMinMeter),
		)
	}
	cfg := RouteSnapConfig(stops, opts...)
	cfg.MaxSnapDistanceMeter = RecommendedMaxSnapDistanceMeter
	cfg.MaxBearingDiffDegree = RecommendedMaxBearingDiffDegree
	cfg.MinMovementMeter = RecommendedMinMovementMeter
	return cfg
}
