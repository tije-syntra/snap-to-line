package snaptoline

import (
	"time"

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
	distReset := s.maybeResetSnapDistance(point)

	candidates := findCandidates(s.segments, point, s.state.LastPoint, s.state, s.config)
	if len(candidates) == 0 {
		if result, best := s.tryHoldLastSegment(point, "no_candidates"); result != nil {
			return s.finishSnap(nil, best, result, point), nil
		}
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

	if stabilized := s.stabilizeSameSegmentCandidate(best, point); stabilized != nil {
		best = stabilized
	}

	if nearby := s.preferNearbyOnActiveSegment(best, point); nearby != nil {
		best = nearby
	}

	if enforced := s.enforceNextStopBeforeSegmentSwitch(best, point); enforced != nil {
		best = enforced
	}

	if locked := s.enforceBranchLock(best, point); locked != nil {
		best = locked
	}

	if s.state.LastBest != nil && s.config.PreventBackwardTransition {
		prevOrder := s.state.LastBest.Segment.Order
		if best.Segment.Order < prevOrder &&
			!isLoopWrapTransition(prevOrder, best.Segment.Order, len(s.segments), s.config.Looping) {
			fallback := s.candidateOnSegment(s.state.LastBest.Segment, point)
			best = &fallback
		}
	}

	result := s.resultFromCandidate(*best, point)

	if s.shouldClampBackward(result) || s.shouldClampMeasureRegression(result) || s.shouldClampLateral(result, point) {
		if clamped := s.clampToPreviousSegment(point); clamped != nil {
			result = clamped
			fallback := s.candidateOnSegment(s.state.LastBest.Segment, point)
			best = &fallback
		}
	}

	if result.SegmentID == "" || result.SegmentOrder <= 0 {
		if held, heldBest := s.tryHoldLastSegment(point, "empty_segment"); held != nil {
			return s.finishSnap(candidates, heldBest, held, point), nil
		}
	}

	if stabilized, stabBest := s.stabilizeWildGPSJump(best, point); stabilized != nil {
		best = stabBest
		result = stabilized
	}

	if capped, capBest := s.capForwardSnapAdvance(best, point); capped != nil {
		best = capBest
		result = capped
	}

	if contResult, contBest := s.applySnapContinuityResult(best, point, result); contResult != nil {
		best = contBest
		result = contResult
	}

	if creepResult, creepBest := s.ensureForwardCreepWhenStuck(best, point); creepResult != nil {
		best = creepBest
		result = creepResult
	}

	result = s.finishSnap(candidates, best, result, point)
	if distReset && result != nil && result.HeldReason == "" {
		result.HeldReason = "snap_distance_reset"
	}
	return result, nil
}

func (s *Snapper) commitSnapState(candidates []Candidate, best *Candidate, point GPSPoint, result *SnapResult) {
	s.state.LastCandidates = candidates
	s.state.LastBest = best
	if best != nil && best.Segment.Order > 0 {
		copy := *best
		s.state.LastGood = &copy
	}
	s.state.LastPoint = &point
	ts := point.Timestamp
	if ts <= 0 {
		ts = time.Now().UnixMilli()
	}
	s.state.LastTimestamp = ts
	if result != nil {
		s.state.LastOutputSnapDistanceM = result.DistanceMeter
	}
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

// PointAtRouteMeasure returns the point on the trip linestring at segment order and progress.
func (s *Snapper) PointAtRouteMeasure(order int, progress float64) orb.Point {
	m := s.RouteMeasure(order, progress)
	p, _ := PointAtMeasure(s.line, m)
	return p
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
