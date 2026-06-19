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
