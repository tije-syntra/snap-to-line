package snaptoline

// Defaults for RouteSnapConfig when RouteSnapParams fields are not set.
const (
	DefaultRouteMeasureRegressionToleranceMeter = 30.0
	DefaultRouteClampBackwardMinConfidence      = 0.55
	DefaultRouteClampDwellSpeedKmh              = 8.0
	DefaultRouteMeasureAdvanceSlackMeter        = 15.0
	DefaultRouteSegmentSwitchHysteresisLog      = 1.0
	DefaultRouteSnappedJumpSlackMeter           = 4.0
	DefaultRouteHoldLastSegmentMaxDistM         = 120.0
	DefaultRouteHoldLastSegmentMaxAgeMs         = int64(30000)
	DefaultRouteHoldLastSegmentMinConfidence    = 0.25
	DefaultRouteWildGPSJumpMinMeter             = 25.0
	DefaultRouteWildGPSJumpMultiplier           = 2.0
	DefaultRouteWildGPSMaxAdvanceFactor         = 0.5
	DefaultRouteMaxForwardSnapMeter             = 0.0 // 0 = no per-tick forward cap
	DefaultRouteNextStopPassToleranceMeter      = 8.0
	DefaultFoldedSegmentMinViable               = 3
	DefaultBranchLockSearchWindowM              = 20.0
	DefaultBranchUnlockNormalTicks              = 3
	DefaultFoldedSegmentMeasureSpreadM          = 45.0
	DefaultSnapDistanceGrowResetTicks           = 2
	DefaultSnapDistanceGrowMinDeltaM            = 8.0
	DefaultSnapDistanceResetMinMeter            = 35.0
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

	// HoldLastSegmentOnMiss reuses previous segment when no snap candidates match.
	// Default: true when nil.
	HoldLastSegmentOnMiss *bool

	// HoldLastSegmentMaxDistM max lateral distance for held projection.
	// Default: DefaultRouteHoldLastSegmentMaxDistM (60) when nil.
	HoldLastSegmentMaxDistM *float64

	// HoldLastSegmentMaxAgeMs max ms since last snap to allow hold.
	// Default: DefaultRouteHoldLastSegmentMaxAgeMs (30000) when nil.
	HoldLastSegmentMaxAgeMs *int64

	// HoldLastSegmentMinConfidence floor confidence on held snaps.
	// Default: DefaultRouteHoldLastSegmentMinConfidence (0.25) when nil.
	HoldLastSegmentMinConfidence *float64

	// WildGPSStabilize freezes or caps snap on implausible raw GPS jumps.
	// Default: true when nil.
	WildGPSStabilize *bool

	// WildGPSJumpMinMeter minimum raw GPS movement for wild-jump detection.
	// Default: DefaultRouteWildGPSJumpMinMeter (25) when nil.
	WildGPSJumpMinMeter *float64

	// WildGPSJumpMultiplier threshold multiplier over plausible movement.
	// Default: DefaultRouteWildGPSJumpMultiplier (2) when nil.
	WildGPSJumpMultiplier *float64

	// WildGPSMaxAdvanceFactor route advance cap factor on wild jumps.
	// Default: DefaultRouteWildGPSMaxAdvanceFactor (0.5) when nil.
	WildGPSMaxAdvanceFactor *float64

	// MaxForwardSnapMeter max route advance per snap along the line.
	// Default: 0 (disabled). Set e.g. 50 to cap forward advance per tick.
	MaxForwardSnapMeter *float64

	// NoBackwardSnap freezes at the last snap when the new result would move backward.
	// Default: true when nil.
	NoBackwardSnap *bool

	// RequireNextStopBeforeSegmentSwitch blocks segment_id changes until the current
	// segment's destination stop is passed. Default: true when nil.
	RequireNextStopBeforeSegmentSwitch *bool

	// NextStopPassToleranceMeter slack before ToMeasure counts as passed.
	// Default: DefaultRouteNextStopPassToleranceMeter (8) when nil.
	NextStopPassToleranceMeter *float64

	// FoldedSegmentBranchLock pins snap on folded geometry (>2 viable projections).
	// Default: true when nil.
	FoldedSegmentBranchLock *bool

	// FoldedSegmentMinViable minimum projections to treat segment as folded (default 3).
	FoldedSegmentMinViable *int

	// BranchLockSearchWindowM measure window while branch lock is active.
	// Default: DefaultBranchLockSearchWindowM (20) when nil.
	BranchLockSearchWindowM *float64

	// BranchUnlockNormalTicks consecutive normal ticks before unlock.
	// Default: DefaultBranchUnlockNormalTicks (3) when nil.
	BranchUnlockNormalTicks *int

	// FoldedSegmentMeasureSpreadM span between viable projections to treat segment as ambiguous.
	// Default: DefaultFoldedSegmentMeasureSpreadM (45) when nil.
	FoldedSegmentMeasureSpreadM *float64

	// SnapContinuityFromPrevious limits snap jumps vs previous snapped position. Default: true when nil.
	SnapContinuityFromPrevious *bool

	// SnapDistanceResetOnGrow resets Viterbi when raw-to-snap distance keeps growing. Default: true when nil.
	SnapDistanceResetOnGrow *bool

	// SnapDistanceGrowResetTicks consecutive growing ticks before reset. Default: 2 when nil.
	SnapDistanceGrowResetTicks *int

	// SnapDistanceGrowMinDeltaM min per-tick distance increase to count as growing. Default: 8 when nil.
	SnapDistanceGrowMinDeltaM *float64

	// SnapDistanceResetMinMeter min raw-to-snap distance before grow-reset applies. Default: 35 when nil.
	SnapDistanceResetMinMeter *float64
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

func WithHoldLastSegmentOnMiss(v bool) RouteSnapOption {
	return func(p *RouteSnapParams) { p.HoldLastSegmentOnMiss = &v }
}

func WithHoldLastSegmentMaxDistM(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.HoldLastSegmentMaxDistM = &m }
}

func WithHoldLastSegmentMaxAgeMs(ms int64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.HoldLastSegmentMaxAgeMs = &ms }
}

func WithHoldLastSegmentMinConfidence(v float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.HoldLastSegmentMinConfidence = &v }
}

func WithWildGPSStabilize(v bool) RouteSnapOption {
	return func(p *RouteSnapParams) { p.WildGPSStabilize = &v }
}

func WithWildGPSJumpMinMeter(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.WildGPSJumpMinMeter = &m }
}

func WithWildGPSJumpMultiplier(v float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.WildGPSJumpMultiplier = &v }
}

func WithWildGPSMaxAdvanceFactor(v float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.WildGPSMaxAdvanceFactor = &v }
}

func WithMaxForwardSnapMeter(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.MaxForwardSnapMeter = &m }
}

func WithNoBackwardSnap(v bool) RouteSnapOption {
	return func(p *RouteSnapParams) { p.NoBackwardSnap = &v }
}

func WithRequireNextStopBeforeSegmentSwitch(v bool) RouteSnapOption {
	return func(p *RouteSnapParams) { p.RequireNextStopBeforeSegmentSwitch = &v }
}

func WithNextStopPassToleranceMeter(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.NextStopPassToleranceMeter = &m }
}

func WithFoldedSegmentBranchLock(v bool) RouteSnapOption {
	return func(p *RouteSnapParams) { p.FoldedSegmentBranchLock = &v }
}

func WithFoldedSegmentMinViable(n int) RouteSnapOption {
	return func(p *RouteSnapParams) { p.FoldedSegmentMinViable = &n }
}

func WithBranchLockSearchWindowM(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.BranchLockSearchWindowM = &m }
}

func WithBranchUnlockNormalTicks(n int) RouteSnapOption {
	return func(p *RouteSnapParams) { p.BranchUnlockNormalTicks = &n }
}

func WithFoldedSegmentMeasureSpreadM(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.FoldedSegmentMeasureSpreadM = &m }
}

func WithSnapContinuityFromPrevious(v bool) RouteSnapOption {
	return func(p *RouteSnapParams) { p.SnapContinuityFromPrevious = &v }
}

func WithSnapDistanceResetOnGrow(v bool) RouteSnapOption {
	return func(p *RouteSnapParams) { p.SnapDistanceResetOnGrow = &v }
}

func WithSnapDistanceGrowResetTicks(n int) RouteSnapOption {
	return func(p *RouteSnapParams) { p.SnapDistanceGrowResetTicks = &n }
}

func WithSnapDistanceGrowMinDeltaM(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.SnapDistanceGrowMinDeltaM = &m }
}

func WithSnapDistanceResetMinMeter(m float64) RouteSnapOption {
	return func(p *RouteSnapParams) { p.SnapDistanceResetMinMeter = &m }
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

	holdLast := true
	if params.HoldLastSegmentOnMiss != nil {
		holdLast = *params.HoldLastSegmentOnMiss
	}
	cfg.HoldLastSegmentOnMiss = holdLast

	holdMaxDist := DefaultRouteHoldLastSegmentMaxDistM
	if params.HoldLastSegmentMaxDistM != nil {
		holdMaxDist = *params.HoldLastSegmentMaxDistM
	}
	cfg.HoldLastSegmentMaxDistM = holdMaxDist

	holdMaxAge := DefaultRouteHoldLastSegmentMaxAgeMs
	if params.HoldLastSegmentMaxAgeMs != nil {
		holdMaxAge = *params.HoldLastSegmentMaxAgeMs
	}
	cfg.HoldLastSegmentMaxAgeMs = holdMaxAge

	holdMinConf := DefaultRouteHoldLastSegmentMinConfidence
	if params.HoldLastSegmentMinConfidence != nil {
		holdMinConf = *params.HoldLastSegmentMinConfidence
	}
	cfg.HoldLastSegmentMinConfidence = holdMinConf

	wildGPS := true
	if params.WildGPSStabilize != nil {
		wildGPS = *params.WildGPSStabilize
	}
	cfg.WildGPSStabilize = wildGPS

	wildMin := DefaultRouteWildGPSJumpMinMeter
	if params.WildGPSJumpMinMeter != nil {
		wildMin = *params.WildGPSJumpMinMeter
	}
	cfg.WildGPSJumpMinMeter = wildMin

	wildMult := DefaultRouteWildGPSJumpMultiplier
	if params.WildGPSJumpMultiplier != nil {
		wildMult = *params.WildGPSJumpMultiplier
	}
	cfg.WildGPSJumpMultiplier = wildMult

	wildAdv := DefaultRouteWildGPSMaxAdvanceFactor
	if params.WildGPSMaxAdvanceFactor != nil {
		wildAdv = *params.WildGPSMaxAdvanceFactor
	}
	cfg.WildGPSMaxAdvanceFactor = wildAdv

	maxFwd := DefaultRouteMaxForwardSnapMeter
	if params.MaxForwardSnapMeter != nil {
		maxFwd = *params.MaxForwardSnapMeter
	}
	cfg.MaxForwardSnapMeter = maxFwd

	noBackward := true
	if params.NoBackwardSnap != nil {
		noBackward = *params.NoBackwardSnap
	}
	cfg.NoBackwardSnap = noBackward

	requireNextStop := true
	if params.RequireNextStopBeforeSegmentSwitch != nil {
		requireNextStop = *params.RequireNextStopBeforeSegmentSwitch
	}
	cfg.RequireNextStopBeforeSegmentSwitch = requireNextStop

	nextStopTol := DefaultRouteNextStopPassToleranceMeter
	if params.NextStopPassToleranceMeter != nil {
		nextStopTol = *params.NextStopPassToleranceMeter
	}
	cfg.NextStopPassToleranceMeter = nextStopTol

	foldedLock := true
	if params.FoldedSegmentBranchLock != nil {
		foldedLock = *params.FoldedSegmentBranchLock
	}
	cfg.FoldedSegmentBranchLock = foldedLock

	foldedMin := DefaultFoldedSegmentMinViable
	if params.FoldedSegmentMinViable != nil {
		foldedMin = *params.FoldedSegmentMinViable
	}
	cfg.FoldedSegmentMinViable = foldedMin

	branchWindow := DefaultBranchLockSearchWindowM
	if params.BranchLockSearchWindowM != nil {
		branchWindow = *params.BranchLockSearchWindowM
	}
	cfg.BranchLockSearchWindowM = branchWindow

	unlockTicks := DefaultBranchUnlockNormalTicks
	if params.BranchUnlockNormalTicks != nil {
		unlockTicks = *params.BranchUnlockNormalTicks
	}
	cfg.BranchUnlockNormalTicks = unlockTicks

	spread := DefaultFoldedSegmentMeasureSpreadM
	if params.FoldedSegmentMeasureSpreadM != nil {
		spread = *params.FoldedSegmentMeasureSpreadM
	}
	cfg.FoldedSegmentMeasureSpreadM = spread

	snapContinuity := true
	if params.SnapContinuityFromPrevious != nil {
		snapContinuity = *params.SnapContinuityFromPrevious
	}
	cfg.SnapContinuityFromPrevious = snapContinuity

	snapDistReset := true
	if params.SnapDistanceResetOnGrow != nil {
		snapDistReset = *params.SnapDistanceResetOnGrow
	}
	cfg.SnapDistanceResetOnGrow = snapDistReset

	growTicks := DefaultSnapDistanceGrowResetTicks
	if params.SnapDistanceGrowResetTicks != nil {
		growTicks = *params.SnapDistanceGrowResetTicks
	}
	cfg.SnapDistanceGrowResetTicks = growTicks

	growDelta := DefaultSnapDistanceGrowMinDeltaM
	if params.SnapDistanceGrowMinDeltaM != nil {
		growDelta = *params.SnapDistanceGrowMinDeltaM
	}
	cfg.SnapDistanceGrowMinDeltaM = growDelta

	resetMin := DefaultSnapDistanceResetMinMeter
	if params.SnapDistanceResetMinMeter != nil {
		resetMin = *params.SnapDistanceResetMinMeter
	}
	cfg.SnapDistanceResetMinMeter = resetMin

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
	if override.HoldLastSegmentOnMiss != nil {
		base.HoldLastSegmentOnMiss = override.HoldLastSegmentOnMiss
	}
	if override.HoldLastSegmentMaxDistM != nil {
		base.HoldLastSegmentMaxDistM = override.HoldLastSegmentMaxDistM
	}
	if override.HoldLastSegmentMaxAgeMs != nil {
		base.HoldLastSegmentMaxAgeMs = override.HoldLastSegmentMaxAgeMs
	}
	if override.HoldLastSegmentMinConfidence != nil {
		base.HoldLastSegmentMinConfidence = override.HoldLastSegmentMinConfidence
	}
	if override.WildGPSStabilize != nil {
		base.WildGPSStabilize = override.WildGPSStabilize
	}
	if override.WildGPSJumpMinMeter != nil {
		base.WildGPSJumpMinMeter = override.WildGPSJumpMinMeter
	}
	if override.WildGPSJumpMultiplier != nil {
		base.WildGPSJumpMultiplier = override.WildGPSJumpMultiplier
	}
	if override.WildGPSMaxAdvanceFactor != nil {
		base.WildGPSMaxAdvanceFactor = override.WildGPSMaxAdvanceFactor
	}
	if override.MaxForwardSnapMeter != nil {
		base.MaxForwardSnapMeter = override.MaxForwardSnapMeter
	}
	if override.NoBackwardSnap != nil {
		base.NoBackwardSnap = override.NoBackwardSnap
	}
	if override.RequireNextStopBeforeSegmentSwitch != nil {
		base.RequireNextStopBeforeSegmentSwitch = override.RequireNextStopBeforeSegmentSwitch
	}
	if override.NextStopPassToleranceMeter != nil {
		base.NextStopPassToleranceMeter = override.NextStopPassToleranceMeter
	}
	if override.FoldedSegmentBranchLock != nil {
		base.FoldedSegmentBranchLock = override.FoldedSegmentBranchLock
	}
	if override.FoldedSegmentMinViable != nil {
		base.FoldedSegmentMinViable = override.FoldedSegmentMinViable
	}
	if override.BranchLockSearchWindowM != nil {
		base.BranchLockSearchWindowM = override.BranchLockSearchWindowM
	}
	if override.BranchUnlockNormalTicks != nil {
		base.BranchUnlockNormalTicks = override.BranchUnlockNormalTicks
	}
	if override.FoldedSegmentMeasureSpreadM != nil {
		base.FoldedSegmentMeasureSpreadM = override.FoldedSegmentMeasureSpreadM
	}
	if override.SnapContinuityFromPrevious != nil {
		base.SnapContinuityFromPrevious = override.SnapContinuityFromPrevious
	}
	if override.SnapDistanceResetOnGrow != nil {
		base.SnapDistanceResetOnGrow = override.SnapDistanceResetOnGrow
	}
	if override.SnapDistanceGrowResetTicks != nil {
		base.SnapDistanceGrowResetTicks = override.SnapDistanceGrowResetTicks
	}
	if override.SnapDistanceGrowMinDeltaM != nil {
		base.SnapDistanceGrowMinDeltaM = override.SnapDistanceGrowMinDeltaM
	}
	if override.SnapDistanceResetMinMeter != nil {
		base.SnapDistanceResetMinMeter = override.SnapDistanceResetMinMeter
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
