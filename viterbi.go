package snaptoline

import (
	"math"
	"sort"

	"github.com/paulmach/orb"
)

type Candidate struct {
	SegmentIndex       int
	Segment            Segment
	Measure            float64
	SnappedPoint       orb.Point
	DistanceMeter      float64
	LineBearing        float64
	EmissionScore      float64
	DirectionScore     float64
	TripDirectionScore float64
	TotalLogScore      float64
	Prev               *Candidate
}

// SegmentDepartLatch tracks halte gate presence so a bus can advance to the next
// segment after departing the junction stop, even when GPS is off-route outside radius.
type SegmentDepartLatch struct {
	GateSegmentOrder int
	WasInsideRadius  bool
	HasDeparted      bool
}

type ViterbiState struct {
	LastCandidates    []Candidate
	LastBest          *Candidate
	LastGood          *Candidate // last snap with a valid segment (for hold after GPS glitches)
	LastPoint         *GPSPoint
	LastTimestamp     int64
	ActiveDirection   DirectionType
	BranchLock        *BranchLock
	BranchNormalTicks int
	RecentBranchTicks []branchTick
	// LastOutputSnapDistanceM is raw GPS to snapped point distance from the previous output.
	LastOutputSnapDistanceM float64
	// GrowingSnapDistTicks counts consecutive ticks where that distance increased.
	GrowingSnapDistTicks int
	SegmentDepart        SegmentDepartLatch
	OffRouteCount        int
	JumpCount            int
	LastGpsJumpRatio     float64
	LastGpsJumpLevel     string
	ReverseCount         int
	TurnaroundValidated  bool
	RecentGpsPoints      []RecentGpsPoint
	LastValidSegmentID   string
	LastValidSegmentOrder int
	LastValidProgress    float64
	LastValidSnappedPoint orb.Point
	SegmentJumpCount     int
	SkippedSegmentCount  int
}

// BranchLock pins snap projection on folded segment geometry.
type BranchLock struct {
	SegmentOrder     int
	SegmentID        string
	LockedRelMeasure float64
	LockedMeasure    float64
	LockedLineIndex  int
	LockedPoint      orb.Point
	ViableCount      int
}

type branchTick struct {
	SegmentOrder int
	LineIndex    int
	Measure      float64
}

func NewViterbiState(activeDirection DirectionType) *ViterbiState {
	return &ViterbiState{ActiveDirection: activeDirection}
}

func (s *ViterbiState) Reset() {
	s.LastCandidates = nil
	s.LastBest = nil
	s.LastGood = nil
	s.LastPoint = nil
	s.LastTimestamp = 0
	s.BranchLock = nil
	s.BranchNormalTicks = 0
	s.RecentBranchTicks = nil
	s.LastOutputSnapDistanceM = 0
	s.GrowingSnapDistTicks = 0
	s.SegmentDepart = SegmentDepartLatch{}
	s.OffRouteCount = 0
	s.JumpCount = 0
	s.LastGpsJumpRatio = 0
	s.LastGpsJumpLevel = ""
	s.ReverseCount = 0
	s.TurnaroundValidated = false
	s.RecentGpsPoints = nil
	s.LastValidSegmentID = ""
	s.LastValidSegmentOrder = 0
	s.LastValidProgress = 0
	s.LastValidSnappedPoint = orb.Point{}
	s.SegmentJumpCount = 0
	s.SkippedSegmentCount = 0
}

func TransitionScore(fromOrder, toOrder, segmentCount int, looping bool) float64 {
	if segmentCount == 0 {
		return 0.05
	}

	fromIdx := fromOrder - 1
	toIdx := toOrder - 1

	if fromIdx == toIdx {
		return 1.0
	}

	if looping && fromIdx == segmentCount-1 && toIdx == 0 {
		return 0.95
	}

	diff := toIdx - fromIdx
	switch {
	case diff == 1:
		return 0.95
	case diff == 2:
		return 0.5
	case diff == -1:
		return 0.15
	case diff < 0:
		return 0.05
	case diff > 2:
		return 0.1
	default:
		return 0.3
	}
}

func findCandidates(
	segments []Segment,
	point GPSPoint,
	prev *GPSPoint,
	state *ViterbiState,
	cfg Config,
) []Candidate {
	type rawCandidate struct {
		segmentIndex int
		projection   ProjectionCandidate
	}

	raw := make([]rawCandidate, 0, len(segments)*2)

	for i, seg := range segments {
		proj := projectOntoSegment(seg, point, prev, state, cfg)
		if proj.DistanceMeter > cfg.MaxSnapDistanceMeter {
			continue
		}

		absMeasure := seg.FromMeasure + proj.Measure
		raw = append(raw, rawCandidate{
			segmentIndex: i,
			projection: ProjectionCandidate{
				Point:         proj.Point,
				Measure:       absMeasure,
				LineIndex:     proj.LineIndex,
				DistanceMeter: proj.DistanceMeter,
			},
		})
	}

	sort.Slice(raw, func(i, j int) bool {
		return raw[i].projection.DistanceMeter < raw[j].projection.DistanceMeter
	})

	if cfg.CandidateLimit > 0 && len(raw) > cfg.CandidateLimit {
		raw = raw[:cfg.CandidateLimit]
	}

	busBearing, hasBearing := resolveBusBearing(point, prev, cfg)
	turnaroundValidated := false
	if state != nil {
		turnaroundValidated = state.TurnaroundValidated
	}
	weaken := shouldWeakenDirectionValidation(point, prev, cfg, turnaroundValidated)

	activeTrip := cfg.TripDirection
	if cfg.UseTripDirection {
		activeTrip = cfg.TripDirection
	} else {
		activeTrip = DirectionUnknown
	}

	candidates := make([]Candidate, 0, len(raw))
	for _, item := range raw {
		seg := segments[item.segmentIndex]
		lineBearing := BearingAtMeasure(seg.Geometry, item.projection.Measure-seg.FromMeasure)
		emission := EmissionScore(item.projection.DistanceMeter, cfg.MaxSnapDistanceMeter)
		dirScore, _ := scoreDirection(busBearing, hasBearing, lineBearing, cfg, weaken)
		tripScore := TripDirectionScore(seg.Direction, activeTrip)

		candidates = append(candidates, Candidate{
			SegmentIndex:       item.segmentIndex,
			Segment:            seg,
			Measure:            item.projection.Measure,
			SnappedPoint:       item.projection.Point,
			DistanceMeter:      item.projection.DistanceMeter,
			LineBearing:        lineBearing,
			EmissionScore:      emission,
			DirectionScore:     dirScore,
			TripDirectionScore: tripScore,
		})
	}

	return candidates
}

func isLoopWrapTransition(fromOrder, toOrder, segmentCount int, looping bool) bool {
	if !looping || segmentCount <= 0 {
		return false
	}
	return fromOrder == segmentCount && toOrder == 1
}

func rejectBackwardCandidate(state *ViterbiState, c Candidate, segmentCount int, point GPSPoint, cfg Config) bool {
	if state.LastBest == nil || !cfg.PreventBackwardTransition {
		return false
	}

	if cfg.ReverseDetection && IsBackwardSnapAllowed(state, cfg) {
		return false
	}

	fromOrder := state.LastBest.Segment.Order
	toOrder := c.Segment.Order
	looping := cfg.Looping
	loopWrap := isLoopWrapTransition(fromOrder, toOrder, segmentCount, looping)

	if toOrder < fromOrder && !loopWrap {
		return true
	}

	tol := cfg.MeasureRegressionToleranceMeter
	if tol > 0 && c.Measure < state.LastBest.Measure-tol && !loopWrap {
		return true
	}

	slack := cfg.MeasureAdvanceSlackMeter
	if slack > 0 && state.LastPoint != nil && !loopWrap {
		movement := DistanceMeter(state.LastPoint.Point, point.Point)
		if movement >= cfg.MinMovementMeter {
			maxAdvance := movement*3 + slack
			if c.Measure > state.LastBest.Measure+maxAdvance {
				return true
			}
		}
	}

	jumpSlack := cfg.SnappedJumpSlackMeter
	if jumpSlack > 0 && state.LastPoint != nil && c.Segment.Order == state.LastBest.Segment.Order {
		jump := DistanceMeter(state.LastBest.SnappedPoint, c.SnappedPoint)
		movement := DistanceMeter(state.LastPoint.Point, point.Point)
		if movement < 1 {
			movement = 1
		}
		if jump > movement*0.75+jumpSlack {
			return true
		}
	}

	return false
}

func viterbiTotalScore(state *ViterbiState, c Candidate, segmentCount int, cfg Config) float64 {
	emissionLog := logScore(c.EmissionScore)
	directionLog := logScore(c.DirectionScore)
	tripLog := logScore(c.TripDirectionScore)
	stepLog := emissionLog + directionLog + tripLog

	if state.LastBest == nil {
		return stepLog
	}

	transition := TransitionScore(state.LastBest.Segment.Order, c.Segment.Order, segmentCount, cfg.Looping)
	return state.LastBest.TotalLogScore + logScore(transition) + stepLog
}

func applySegmentSwitchHysteresis(
	state *ViterbiState,
	best *Candidate,
	candidates []Candidate,
	segmentCount int,
	point GPSPoint,
	cfg Config,
) *Candidate {
	if state.LastBest == nil || best == nil {
		return best
	}

	hyst := cfg.SegmentSwitchHysteresisLog
	if hyst <= 0 {
		return best
	}

	prevOrder := state.LastBest.Segment.Order
	if best.Segment.Order == prevOrder {
		return best
	}

	var prevBest *Candidate
	for i := range candidates {
		c := candidates[i]
		if c.Segment.Order != prevOrder {
			continue
		}
		if rejectBackwardCandidate(state, c, segmentCount, point, cfg) {
			continue
		}
		copy := c
		copy.TotalLogScore = viterbiTotalScore(state, copy, segmentCount, cfg)
		if prevBest == nil || copy.TotalLogScore > prevBest.TotalLogScore {
			prevBest = &copy
		}
	}

	if prevBest == nil {
		return best
	}

	// Only hesitate when both segments project nearby (parallel overlap ambiguity).
	const ambiguousDistM = 12.0
	if best.DistanceMeter > ambiguousDistM || prevBest.DistanceMeter > ambiguousDistM {
		return best
	}
	if math.Abs(best.DistanceMeter-prevBest.DistanceMeter) > 5 {
		return best
	}

	if best.TotalLogScore-prevBest.TotalLogScore < hyst {
		return prevBest
	}
	return best
}

func runViterbiStep(
	state *ViterbiState,
	candidates []Candidate,
	segmentCount int,
	point GPSPoint,
	cfg Config,
) *Candidate {
	if len(candidates) == 0 {
		return nil
	}

	var best *Candidate
	looping := cfg.Looping

	for i := range candidates {
		c := candidates[i]
		if rejectBackwardCandidate(state, c, segmentCount, point, cfg) {
			continue
		}
		if RejectSegmentJumpCandidate(state, c, segmentCount, cfg) {
			continue
		}
		if rejectSegmentSwitch(state, c, segmentCount, point, cfg) &&
			(cfg.RequireNextStopBeforeSegmentSwitch || cfg.RequireStopRadiusForSegmentSwitch) {
			continue
		}

		emissionLog := logScore(c.EmissionScore)
		directionLog := logScore(c.DirectionScore)
		tripLog := logScore(c.TripDirectionScore)
		stepLog := emissionLog + directionLog + tripLog

		if state.LastBest == nil {
			total := stepLog
			candidate := c
			candidate.TotalLogScore = total
			if best == nil || candidate.TotalLogScore > best.TotalLogScore {
				copy := candidate
				best = &copy
			}
			continue
		}

		transition := TransitionScore(state.LastBest.Segment.Order, c.Segment.Order, segmentCount, looping)
		total := state.LastBest.TotalLogScore + logScore(transition) + stepLog

		candidate := c
		candidate.TotalLogScore = total
		prev := *state.LastBest
		candidate.Prev = &prev

		if best == nil || candidate.TotalLogScore > best.TotalLogScore {
			copy := candidate
			best = &copy
		}
	}

	best = applySegmentSwitchHysteresis(state, best, candidates, segmentCount, point, cfg)

	if best == nil && state.LastBest != nil && cfg.PreventBackwardTransition {
		for i := range candidates {
			c := candidates[i]
			if c.Segment.Order == state.LastBest.Segment.Order {
				copy := c
				return &copy
			}
		}
		if picked := pickNearestForwardCandidate(candidates, state, segmentCount, looping); picked != nil {
			return picked
		}
	}

	return best
}

func confidenceFromScores(c Candidate) float64 {
	emission := c.EmissionScore
	direction := c.DirectionScore
	trip := c.TripDirectionScore

	score := emission * direction * trip
	if score > 1 {
		return 1
	}
	if score < 0 {
		return 0
	}
	return score
}

func clampConfidence(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}
