package snaptoline

import (
	"fmt"
	"sort"

	"github.com/paulmach/orb"
	"github.com/tije-syntra/snap-to-line/internal/mathgeo"
)

func IsSameStop(a, b Stop, toleranceMeter float64) bool {
	if a.ID != b.ID {
		return false
	}
	return DistanceMeter(a.Point, b.Point) <= toleranceMeter
}

func sortStopsByOrder(stops []Stop) []Stop {
	sorted := make([]Stop, len(stops))
	copy(sorted, stops)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Order < sorted[j].Order
	})
	return sorted
}

func CountOccurrence(projected []ProjectedStop, stopID string) int {
	count := 0
	for _, p := range projected {
		if p.Stop.ID == stopID {
			count++
		}
	}
	return count
}

func ProjectStopNearMeasure(line orb.LineString, stop Stop, targetMeasure float64, cfg Config) ProjectedStop {
	total := LineLengthMeter(line)
	window := total * 0.15
	if window < 50 {
		window = 50
	}
	if window > 500 {
		window = 500
	}

	candidate := mathgeo.FindProjectionNearMeasure(line, stop.Point, targetMeasure, window)
	return ProjectedStop{
		Stop:      stop,
		Measure:   candidate.Measure,
		LineIndex: candidate.LineIndex,
	}
}

func ProjectStopForwardOnly(line orb.LineString, stop Stop, minMeasure float64, cfg Config) ProjectedStop {
	candidates := FindProjectionCandidates(line, stop.Point)

	var best *ProjectionCandidate
	for i := range candidates {
		c := &candidates[i]
		if c.Measure < minMeasure {
			continue
		}
		if best == nil || c.DistanceMeter < best.DistanceMeter {
			best = c
		}
	}

	if best == nil {
		fallback := FindNearestProjectionAfterMeasure(line, stop.Point, minMeasure)
		best = &fallback
	}

	return ProjectedStop{
		Stop:      stop,
		Measure:   best.Measure,
		LineIndex: best.LineIndex,
	}
}

func ProjectStopsSequential(line orb.LineString, stops []Stop, cfg Config) ([]ProjectedStop, error) {
	if len(stops) < 2 {
		return nil, ErrInsufficientStops
	}
	if len(line) < 2 {
		return nil, ErrInvalidLine
	}

	stops = sortStopsByOrder(stops)
	totalLength := LineLengthMeter(line)
	result := make([]ProjectedStop, 0, len(stops))

	firstStop := stops[0]
	lastStop := stops[len(stops)-1]

	isClosedLoopStop := cfg.Looping &&
		cfg.AllowSameStartEndStop &&
		IsSameStop(firstStop, lastStop, cfg.LoopClosureToleranceMeter)

	lastMeasure := 0.0

	for i, stop := range stops {
		var projected ProjectedStop

		switch {
		case i == 0:
			projected = ProjectStopNearMeasure(line, stop, 0, cfg)
			projected.Measure = 0
			projected.Occurrence = 1
		case isClosedLoopStop && i == len(stops)-1:
			projected = ProjectStopNearMeasure(line, stop, totalLength, cfg)
			projected.Measure = totalLength
			projected.Occurrence = 2
			projected.IsLoopClosure = true
		default:
			projected = ProjectStopForwardOnly(line, stop, lastMeasure, cfg)
			projected.Occurrence = CountOccurrence(result, stop.ID) + 1
		}

		if projected.Measure < lastMeasure {
			return nil, ErrStopMeasureNotMonotonic
		}

		result = append(result, projected)
		lastMeasure = projected.Measure
	}

	return result, nil
}

func DetectLoopClosure(projected []ProjectedStop) bool {
	if len(projected) < 2 {
		return false
	}
	return projected[len(projected)-1].IsLoopClosure
}

func segmentID(from, to Stop, order int) string {
	return fmt.Sprintf("SEG-%s-%s-%d", from.ID, to.ID, order)
}
