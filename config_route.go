package snaptoline

// Defaults for RouteSnapConfig when RouteSnapParams fields are not set.
const (
	DefaultRouteMeasureRegressionToleranceMeter = 30.0
	DefaultRouteClampBackwardMinConfidence      = 0.55
	DefaultRouteClampDwellSpeedKmh              = 8.0
	DefaultRouteMeasureAdvanceSlackMeter        = 15.0
	DefaultRouteSegmentSwitchHysteresisLog      = 1.0
	DefaultRouteSnappedJumpSlackMeter           = 4.0
)

// RouteSnapParams holds optional overrides for RouteSnapConfig.
// Use nil pointer fields to keep the default for that setting.
// Pass no params (or zero-value RouteSnapParams) to use all defaults.
//
// Example:
//
//	cfg := RouteSnapConfig(stops, RouteSnapParams{
//	    MeasureRegressionToleranceMeter: ptr(40.0),
//	    ClampDwellSpeedKmh:              ptr(5.0),
//	})
type RouteSnapParams struct {
	// PreventBackwardTransition rejects Viterbi candidates on a lower segment order.
	// Default: true when nil.
	PreventBackwardTransition *bool

	// MeasureRegressionToleranceMeter rejects snaps whose route measure drops more than this.
	// Default: DefaultRouteMeasureRegressionToleranceMeter (30) when nil.
	MeasureRegressionToleranceMeter *float64

	// ClampBackwardMinConfidence enables post-Viterbi backward clamp below this confidence.
	// Default: DefaultRouteClampBackwardMinConfidence (0.55) when nil.
	// Set explicitly to 0 to disable clamp.
	ClampBackwardMinConfidence *float64

	// ClampDwellSpeedKmh treats speed at or below this as dwell when clamping.
	// Default: DefaultRouteClampDwellSpeedKmh (8) when nil.
	ClampDwellSpeedKmh *float64

	// MeasureAdvanceSlackMeter caps forward measure jumps on overlapping geometry.
	// Default: DefaultRouteMeasureAdvanceSlackMeter (15) when nil. Set to 0 to disable.
	MeasureAdvanceSlackMeter *float64

	// SegmentSwitchHysteresisLog minimum log-score margin to change segment order.
	// Default: DefaultRouteSegmentSwitchHysteresisLog (1.0) when nil. Set to 0 to disable.
	SegmentSwitchHysteresisLog *float64

	// SnappedJumpSlackMeter caps lateral snap jumps relative to GPS movement on folded geometry.
	// Default: DefaultRouteSnappedJumpSlackMeter (4) when nil. Set to 0 to disable.
	SnappedJumpSlackMeter *float64

	// Looping enables loop-route handling. Default: auto-detect from stops when nil
	// (true when first and last stop are the same within LoopClosureToleranceMeter).
	Looping *bool

	// LoopClosureToleranceMeter used for same start/end stop detection when Looping is auto.
	// Default: DefaultConfig().LoopClosureToleranceMeter (10) when nil.
	LoopClosureToleranceMeter *float64
}

// RouteSnapOption configures RouteSnapConfig. Prefer helper functions below or RouteSnapParamsOption.
type RouteSnapOption func(*RouteSnapParams)

// RouteSnapParamsOption applies a params struct as a functional option.
func RouteSnapParamsOption(p RouteSnapParams) RouteSnapOption {
	return func(acc *RouteSnapParams) {
		*acc = mergeRouteSnapParams(*acc, p)
	}
}

func WithRouteSnapParams(p RouteSnapParams) RouteSnapOption {
	return RouteSnapParamsOption(p)
}

func WithPreventBackwardTransition(v bool) RouteSnapOption {
	return func(p *RouteSnapParams) { p.PreventBackwardTransition = &v }
}

func WithMeasureRegressionTolerance(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.MeasureRegressionToleranceMeter = &m }
}

func WithClampBackwardMinConfidence(v float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.ClampBackwardMinConfidence = &v }
}

func WithClampDwellSpeedKmh(kmh float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.ClampDwellSpeedKmh = &kmh }
}

func WithMeasureAdvanceSlack(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.MeasureAdvanceSlackMeter = &m }
}

func WithSnappedJumpSlack(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.SnappedJumpSlackMeter = &m }
}

func WithSegmentSwitchHysteresisLog(v float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.SegmentSwitchHysteresisLog = &v }
}

func WithLooping(v bool) RouteSnapOption {
	return func(p *RouteSnapParams) { p.Looping = &v }
}

func WithLoopClosureTolerance(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.LoopClosureToleranceMeter = &m }
}

// DisableBackwardClamp sets ClampBackwardMinConfidence to 0 (disables post-Viterbi clamp).
func DisableBackwardClamp() RouteSnapOption {
	zero := 0.0
	return func(p *RouteSnapParams) { p.ClampBackwardMinConfidence = &zero }
}

// RouteSnapConfig returns snap settings tuned for live bus tracking on gate-to-gate routes.
// Call with no options to use defaults. Options are applied in order; later options override earlier ones.
func RouteSnapConfig(stops []Stop, opts ...RouteSnapOption) Config {
	params := RouteSnapParams{}
	for _, opt := range opts {
		if opt != nil {
			opt(&params)
		}
	}
	return routeSnapConfig(stops, params)
}

func routeSnapConfig(stops []Stop, params RouteSnapParams) Config {
	cfg := DefaultConfig()

	preventBackward := true
	if params.PreventBackwardTransition != nil {
		preventBackward = *params.PreventBackwardTransition
	}
	cfg.PreventBackwardTransition = preventBackward

	measureTol := DefaultRouteMeasureRegressionToleranceMeter
	if params.MeasureRegressionToleranceMeter != nil {
		measureTol = *params.MeasureRegressionToleranceMeter
	}
	cfg.MeasureRegressionToleranceMeter = measureTol

	clampConf := DefaultRouteClampBackwardMinConfidence
	if params.ClampBackwardMinConfidence != nil {
		clampConf = *params.ClampBackwardMinConfidence
	}
	cfg.ClampBackwardMinConfidence = clampConf

	clampDwell := DefaultRouteClampDwellSpeedKmh
	if params.ClampDwellSpeedKmh != nil {
		clampDwell = *params.ClampDwellSpeedKmh
	}
	cfg.ClampDwellSpeedKmh = clampDwell

	advanceSlack := DefaultRouteMeasureAdvanceSlackMeter
	if params.MeasureAdvanceSlackMeter != nil {
		advanceSlack = *params.MeasureAdvanceSlackMeter
	}
	cfg.MeasureAdvanceSlackMeter = advanceSlack

	switchHyst := DefaultRouteSegmentSwitchHysteresisLog
	if params.SegmentSwitchHysteresisLog != nil {
		switchHyst = *params.SegmentSwitchHysteresisLog
	}
	cfg.SegmentSwitchHysteresisLog = switchHyst

	jumpSlack := DefaultRouteSnappedJumpSlackMeter
	if params.SnappedJumpSlackMeter != nil {
		jumpSlack = *params.SnappedJumpSlackMeter
	}
	cfg.SnappedJumpSlackMeter = jumpSlack

	loopTol := cfg.LoopClosureToleranceMeter
	if params.LoopClosureToleranceMeter != nil {
		loopTol = *params.LoopClosureToleranceMeter
		cfg.LoopClosureToleranceMeter = loopTol
	}

	if params.Looping != nil {
		cfg.Looping = *params.Looping
	} else {
		cfg.Looping = detectLoopFromStops(stops, cfg.AllowSameStartEndStop, loopTol)
	}

	return cfg
}

func mergeRouteSnapParams(base, override RouteSnapParams) RouteSnapParams {
	if override.PreventBackwardTransition != nil {
		base.PreventBackwardTransition = override.PreventBackwardTransition
	}
	if override.MeasureRegressionToleranceMeter != nil {
		base.MeasureRegressionToleranceMeter = override.MeasureRegressionToleranceMeter
	}
	if override.ClampBackwardMinConfidence != nil {
		base.ClampBackwardMinConfidence = override.ClampBackwardMinConfidence
	}
	if override.ClampDwellSpeedKmh != nil {
		base.ClampDwellSpeedKmh = override.ClampDwellSpeedKmh
	}
	if override.MeasureAdvanceSlackMeter != nil {
		base.MeasureAdvanceSlackMeter = override.MeasureAdvanceSlackMeter
	}
	if override.SegmentSwitchHysteresisLog != nil {
		base.SegmentSwitchHysteresisLog = override.SegmentSwitchHysteresisLog
	}
	if override.SnappedJumpSlackMeter != nil {
		base.SnappedJumpSlackMeter = override.SnappedJumpSlackMeter
	}
	if override.Looping != nil {
		base.Looping = override.Looping
	}
	if override.LoopClosureToleranceMeter != nil {
		base.LoopClosureToleranceMeter = override.LoopClosureToleranceMeter
	}
	return base
}

func detectLoopFromStops(stops []Stop, allowSame bool, toleranceM float64) bool {
	if !allowSame {
		return false
	}
	sorted := sortStopsByOrder(stops)
	if len(sorted) < 2 {
		return false
	}
	first, last := sorted[0], sorted[len(sorted)-1]
	return IsSameStop(first, last, toleranceM)
}
