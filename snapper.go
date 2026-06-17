package snaptoline

import (
	"github.com/paulmach/orb"
)

type Snapper struct {
	line     orb.LineString
	segments []Segment
	stops    []Stop
	config   Config
	state    *ViterbiState
}

func NewSnapper(line orb.LineString, stops []Stop, cfg Config) (*Snapper, error) {
	if len(line) < 2 {
		return nil, ErrInvalidLine
	}
	if len(stops) < 2 {
		return nil, ErrInsufficientStops
	}

	sortedStops := sortStopsByOrder(stops)
	segments, err := BuildSegments(line, sortedStops, cfg)
	if err != nil {
		return nil, err
	}
	if len(segments) == 0 {
		return nil, ErrEmptySegments
	}

	activeDirection := DirectionUnknown
	if cfg.UseTripDirection {
		activeDirection = cfg.TripDirection
	}

	return &Snapper{
		line:     line,
		segments: segments,
		stops:    sortedStops,
		config:   cfg,
		state:    NewViterbiState(activeDirection),
	}, nil
}

func (s *Snapper) Snap(point GPSPoint) (*SnapResult, error) {
	candidates := findCandidates(s.segments, point, s.state.LastPoint, s.config)
	if len(candidates) == 0 {
		return &SnapResult{
			OriginalPoint:  point.Point,
			SnappedPoint:   point.Point,
			IsOffRoute:     true,
			RejectedReason: "no candidates within max snap distance",
		}, nil
	}

	best := runViterbiStep(s.state, candidates, len(s.segments), s.config.Looping)
	if best == nil {
		return nil, ErrNoCandidates
	}

	busBearing, hasBearing := resolveBusBearing(point, s.state.LastPoint, s.config)
	weaken := shouldWeakenDirectionValidation(point, s.state.LastPoint, s.config)
	_, directionDiff := scoreDirection(busBearing, hasBearing, best.LineBearing, s.config, weaken)

	if !hasBearing && s.state.LastBest != nil {
		busBearing = s.state.LastBest.LineBearing
	}

	result := &SnapResult{
		OriginalPoint: point.Point,
		SnappedPoint:  best.SnappedPoint,
		SegmentID:     best.Segment.ID,
		SegmentOrder:  best.Segment.Order,
		Direction:     best.Segment.Direction,
		NearestStopID: nearestStopID(s.stops, best.SnappedPoint),
		DistanceMeter: best.DistanceMeter,
		Progress:      segmentProgress(best.Segment, best.Measure),
		BusBearing:    busBearing,
		LineBearing:   best.LineBearing,
		DirectionDiff: directionDiff,
		Confidence:    clampConfidence(confidenceFromScores(*best)),
		IsOffRoute:    best.DistanceMeter > s.config.MaxSnapDistanceMeter,
	}

	s.state.LastCandidates = candidates
	s.state.LastBest = best
	s.state.LastPoint = &point
	s.state.LastTimestamp = point.Timestamp

	return result, nil
}

func (s *Snapper) Reset() {
	s.state.Reset()
}

func (s *Snapper) SetTripDirection(direction DirectionType) {
	s.config.TripDirection = direction
	s.config.UseTripDirection = direction != DirectionUnknown
	s.state.ActiveDirection = direction
}

func (s *Snapper) Segments() []Segment {
	out := make([]Segment, len(s.segments))
	copy(out, s.segments)
	return out
}

func (s *Snapper) Config() Config {
	return s.config
}
