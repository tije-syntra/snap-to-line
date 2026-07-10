package snaptoline

import "github.com/paulmach/orb"

type DirectionType string

const (
	DirectionOutbound DirectionType = "outbound"
	DirectionInbound  DirectionType = "inbound"
	DirectionLoop     DirectionType = "loop"
	DirectionUnknown  DirectionType = "unknown"
)

type Stop struct {
	ID    string
	Name  string
	Point orb.Point
	Order int
}

type ProjectedStop struct {
	Stop          Stop
	Measure       float64
	LineIndex     int
	Occurrence    int
	IsLoopClosure bool
}

type Segment struct {
	ID            string
	FromStop      Stop
	ToStop        Stop
	FromMeasure   float64
	ToMeasure     float64
	Geometry      orb.LineString
	Order         int
	Direction     DirectionType
	Bearing       float64
	IsLooping     bool
	IsLoopClosing bool
}

type GPSPoint struct {
	Point     orb.Point
	Speed     float64
	Bearing   float64
	Timestamp int64
}

type SnapResult struct {
	OriginalPoint  orb.Point
	SnappedPoint   orb.Point
	SegmentID      string
	SegmentOrder   int
	Direction      DirectionType
	NearestStopID  string
	DistanceMeter  float64
	Progress       float64
	BusBearing     float64
	LineBearing    float64
	DirectionDiff  float64
	Confidence     float64
	IsOffRoute     bool
	RejectedReason string
	// HeldSegment is true when segment/order were kept from the previous snap (e.g. no candidates).
	HeldSegment bool
	HeldReason  string
	// BranchLocked is true when snap is pinned to a branch on folded segment geometry.
	BranchLocked bool
	// OffRouteCount is consecutive samples beyond OffRouteDistanceMeter (when OffRouteDetection is on).
	OffRouteCount int
	// JumpCount counts GPS movements exceeding GpsJumpCountDistanceMeter.
	JumpCount    int
	GpsJumpRatio float64
	GpsJumpLevel string
	// ReverseCount counts consecutive backward measure samples.
	ReverseCount        int
	TurnaroundValidated bool
	SegmentJumpCount    int
	SkippedSegmentCount int
}

type Config struct {
	MaxSnapDistanceMeter float64
	CandidateLimit       int

	Looping bool

	UseBearingValidation bool
	MaxBearingDiffDegree float64

	UseMovementBearing bool
	MinMovementMeter   float64

	UseTripDirection bool
	TripDirection    DirectionType

	AllowSameStartEndStop     bool
	LoopClosureToleranceMeter float64

	UseSpeed bool

	// SegmentDirections optionally overrides direction per segment (index matches segment order).
	SegmentDirections []DirectionType

	// PreventBackwardTransition rejects Viterbi candidates on a lower segment order
	// (except loop wrap last→first when Looping is true).
	PreventBackwardTransition bool

	// MeasureRegressionToleranceMeter rejects candidates whose route measure drops
	// more than this below the previous snap (except loop wrap). Zero disables.
	MeasureRegressionToleranceMeter float64

	// ClampBackwardMinConfidence enables post-Viterbi clamp when a backward segment
	// slip occurs with confidence below this (0 disables clamp). Used with PreventBackwardTransition.
	ClampBackwardMinConfidence float64

	// ClampDwellSpeedKmh treats snaps at or below this speed as dwell when deciding clamp.
	ClampDwellSpeedKmh float64

	// MeasureAdvanceSlackMeter caps unrealistic forward measure jumps on folded/overlapping
	// geometry: max advance = GPS movement * 3 + slack. Zero disables.
	MeasureAdvanceSlackMeter float64

	// SegmentSwitchHysteresisLog minimum log-score margin to change segment order.
	// Zero disables. Prefer staying on the current segment when ambiguous.
	SegmentSwitchHysteresisLog float64

	// SnappedJumpSlackMeter caps lateral snap jumps vs GPS movement on overlapping geometry.
	// Zero disables.
	SnappedJumpSlackMeter float64

	// HoldLastSegmentOnMiss reuses the previous segment when no candidates are within
	// MaxSnapDistanceMeter (live-bus GPS glitches). Disabled when false.
	HoldLastSegmentOnMiss bool

	// HoldLastSegmentMaxDistM max lateral distance for held projection (defaults in RouteSnapConfig).
	HoldLastSegmentMaxDistM float64

	// HoldLastSegmentMaxAgeMs max elapsed ms since the last good snap to allow hold. Zero = use default.
	HoldLastSegmentMaxAgeMs int64

	// HoldLastSegmentMinConfidence floor for confidence on held snaps (ETA consumers).
	HoldLastSegmentMinConfidence float64

	// WildGPSStabilize freezes or caps snap when raw GPS jumps implausibly far.
	WildGPSStabilize bool

	// WildGPSJumpMinMeter minimum raw GPS movement before wild-jump detection applies.
	WildGPSJumpMinMeter float64

	// WildGPSJumpMultiplier raw movement above plausible speed*time*multiplier is wild.
	WildGPSJumpMultiplier float64

	// WildGPSMaxAdvanceFactor max route advance on wild jump = rawMovement*factor + slack.
	WildGPSMaxAdvanceFactor float64

	// MaxForwardSnapMeter max route measure advance per snap along the line (0 disables).
	MaxForwardSnapMeter float64

	// NoBackwardSnap rejects any snap whose route measure or segment order moves backward.
	NoBackwardSnap bool

	// RequireNextStopBeforeSegmentSwitch blocks segment_id changes until the bus has
	// passed the current segment's destination stop (ToStop).
	RequireNextStopBeforeSegmentSwitch bool

	// NextStopPassToleranceMeter route measure slack before ToMeasure counts as passed.
	NextStopPassToleranceMeter float64

	// RequireStopRadiusForSegmentSwitch allows segment order changes only when raw GPS
	// is within SegmentSwitchStopRadiusMeter of the segment junction stop (gate halte).
	RequireStopRadiusForSegmentSwitch bool

	// SegmentSwitchStopRadiusMeter max distance from junction stop to allow segment switch.
	// Zero uses DefaultRouteSegmentSwitchStopRadiusMeter when RequireStopRadiusForSegmentSwitch is true.
	SegmentSwitchStopRadiusMeter float64

	// FoldedSegmentBranchLock pins snap to the nearest GPS branch when a segment has
	// more than FoldedSegmentMinViable projections within max snap distance.
	FoldedSegmentBranchLock bool

	// FoldedSegmentMinViable minimum viable projections to treat a segment as folded (>2 → 3).
	FoldedSegmentMinViable int

	// BranchLockSearchWindowM relative measure window while a branch lock is active.
	BranchLockSearchWindowM float64

	// BranchUnlockNormalTicks consecutive ticks with ≤2 viable projections before unlock.
	BranchUnlockNormalTicks int

	// FoldedSegmentMeasureSpreadM treats a segment as ambiguous when viable projection
	// measures span more than this distance (overlapping geometry with few candidates).
	FoldedSegmentMeasureSpreadM float64

	// SnapContinuityFromPrevious limits snap jumps vs the previous snapped position.
	SnapContinuityFromPrevious bool

	// SnapDistanceResetOnGrow clears Viterbi state when raw-to-snap distance keeps growing.
	SnapDistanceResetOnGrow bool

	// SnapDistanceGrowResetTicks consecutive growing-distance ticks before reset (default 2).
	SnapDistanceGrowResetTicks int

	// SnapDistanceGrowMinDeltaM minimum distance increase per tick to count as growing (default 8).
	SnapDistanceGrowMinDeltaM float64

	// SnapDistanceResetMinMeter minimum raw-to-snap distance before grow-reset applies (default 35).
	SnapDistanceResetMinMeter float64

	// SnapDistanceResetMaxMeter immediate reset when raw-to-snap distance reaches this (default 100, 0 = off).
	SnapDistanceResetMaxMeter float64

	// TeleportDetection rejects implausibly fast GPS movement within TeleportTimeSec.
	TeleportDetection        bool
	TeleportDistanceMeter    float64
	TeleportTimeSec          float64
	TeleportSpeedMatchFactor float64

	// OffRouteDetection flags off-route after OffRouteConsecutiveSamples beyond OffRouteDistanceMeter.
	OffRouteDetection          bool
	OffRouteDistanceMeter      float64
	OffRouteConsecutiveSamples int

	// GpsJumpDetection classifies GPS jumps by expected-vs-actual distance ratio.
	GpsJumpDetection          bool
	GpsJumpExpectedFactor     float64
	GpsJumpMinExpectedMeter   float64
	GpsJumpWarningRatio       float64
	GpsJumpSuspiciousRatio    float64
	GpsJumpRejectRatio        float64
	GpsJumpCountDistanceMeter float64

	// ReverseDetection holds backward measure movement with turnaround validation.
	ReverseDetection                  bool
	ReverseMeasureEpsilonMeter        float64
	ReverseAcceptAfterSamples         int
	ReverseIgnoreMeter                float64
	ReverseHoldMeter                  float64
	ReverseWarningMeter               float64
	ReverseMinSpeedKmh                float64
	ReverseTurnDetection              bool
	ReverseTurnSampleWindow           int
	ReverseTurnMinMovementMeter       float64
	ReverseTurnMinMovementAngleDegree float64
	ReverseTurnCumulativeAngleDegree  float64
	ReverseTurnRouteOppositionDegree  float64

	// SegmentSequenceValidation rejects or recovers invalid segment order jumps.
	SegmentSequenceValidation                 bool
	SegmentJumpRecoverySamples                int
	SegmentJumpRecoveryMinConfidence          float64
	SegmentJumpRecoveryMaxDistanceMeter       float64
	SegmentJumpRecoveryMaxDirectionDiffDegree float64
	SegmentJumpRecoveryGpsFactor              float64
}

func DefaultConfig() Config {
	return Config{
		MaxSnapDistanceMeter:      60,
		CandidateLimit:            8,
		Looping:                   false,
		UseBearingValidation:      true,
		MaxBearingDiffDegree:      60,
		UseMovementBearing:        true,
		MinMovementMeter:          8,
		UseTripDirection:          false,
		TripDirection:             DirectionUnknown,
		AllowSameStartEndStop:     true,
		LoopClosureToleranceMeter: 10,
		UseSpeed:                  true,
	}
}
