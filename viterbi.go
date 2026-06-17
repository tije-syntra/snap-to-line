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

type ViterbiState struct {
	LastCandidates  []Candidate
	LastBest        *Candidate
	LastPoint       *GPSPoint
	LastTimestamp   int64
	ActiveDirection DirectionType
}

func NewViterbiState(activeDirection DirectionType) *ViterbiState {
	return &ViterbiState{ActiveDirection: activeDirection}
}

func (s *ViterbiState) Reset() {
	s.LastCandidates = nil
	s.LastBest = nil
	s.LastPoint = nil
	s.LastTimestamp = 0
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
	cfg Config,
) []Candidate {
	type rawCandidate struct {
		segmentIndex int
		projection   ProjectionCandidate
	}

	raw := make([]rawCandidate, 0, len(segments)*2)

	for i, seg := range segments {
		proj := ProjectPointOnLine(seg.Geometry, point.Point)
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
	weaken := shouldWeakenDirectionValidation(point, prev, cfg)

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

func runViterbiStep(
	state *ViterbiState,
	candidates []Candidate,
	segmentCount int,
	looping bool,
) *Candidate {
	if len(candidates) == 0 {
		return nil
	}

	var best *Candidate

	for i := range candidates {
		c := candidates[i]
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
