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
	candidates := findCandidates(s.segments, point, s.state.LastPoint, s.state, s.config)
	if len(candidates) == 0 {
		return &SnapResult{
			OriginalPoint:  point.Point,
			SnappedPoint:   point.Point,
			IsOffRoute:     true,
			RejectedReason: "no candidates within max snap distance",
		}, nil
	}

	best := runViterbiStep(s.state, candidates, len(s.segments), point, s.config)
	if best == nil {
		if s.state.LastBest != nil && s.config.PreventBackwardTransition {
			fallback := s.candidateOnSegment(s.state.LastBest.Segment, point)
			best = &fallback
		} else {
			return nil, ErrNoCandidates
		}
	}

	result := s.resultFromCandidate(*best, point)

	if s.shouldClampBackward(result, point) || s.shouldClampOverlap(result, point) || s.shouldClampLateral(result, point) {
		if clamped := s.clampToPreviousSegment(point); clamped != nil {
			result = clamped
			fallback := s.candidateOnSegment(s.state.LastBest.Segment, point)
			best = &fallback
		}
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

// RouteMeasure returns cumulative distance along the route for a segment order and progress.
func (s *Snapper) RouteMeasure(order int, progress float64) float64 {
	for _, seg := range s.segments {
		if seg.Order == order {
			return seg.FromMeasure + progress*(seg.ToMeasure-seg.FromMeasure)
		}
	}
	return 0
}

// SnapResultFromSegment projects a GPS point onto a specific segment and builds
// a SnapResult without updating Viterbi state.
func SnapResultFromSegment(seg Segment, stops []Stop, point GPSPoint, prev *GPSPoint, cfg Config) *SnapResult {
	proj := ProjectPointOnLine(seg.Geometry, point.Point)
	absMeasure := seg.FromMeasure + proj.Measure
	lineBearing := BearingAtMeasure(seg.Geometry, proj.Measure)

	busBearing, hasBearing := resolveBusBearing(point, prev, cfg)
	weaken := shouldWeakenDirectionValidation(point, prev, cfg)
	_, directionDiff := scoreDirection(busBearing, hasBearing, lineBearing, cfg, weaken)
	if !hasBearing {
		busBearing = lineBearing
	}

	emission := EmissionScore(proj.DistanceMeter, cfg.MaxSnapDistanceMeter)
	dirScore, _ := scoreDirection(busBearing, hasBearing, lineBearing, cfg, weaken)
	tripScore := TripDirectionScore(seg.Direction, cfg.TripDirection)
	confidence := clampConfidence(emission * dirScore * tripScore)

	return &SnapResult{
		OriginalPoint: point.Point,
		SnappedPoint:  proj.Point,
		SegmentID:     seg.ID,
		SegmentOrder:  seg.Order,
		Direction:     seg.Direction,
		NearestStopID: nearestStopID(stops, proj.Point),
		DistanceMeter: proj.DistanceMeter,
		Progress:      segmentProgress(seg, absMeasure),
		BusBearing:    busBearing,
		LineBearing:   lineBearing,
		DirectionDiff: directionDiff,
		Confidence:    confidence,
		IsOffRoute:    proj.DistanceMeter > cfg.MaxSnapDistanceMeter,
	}
}
