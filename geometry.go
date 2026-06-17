package snaptoline

import (
	"errors"

	"github.com/paulmach/orb"
	"github.com/tije-syntra/snap-to-line/internal/mathgeo"
)

func DistanceMeter(a, b orb.Point) float64 {
	return mathgeo.DistanceMeter(a, b)
}

func LineLengthMeter(line orb.LineString) float64 {
	return mathgeo.LineLengthMeter(line)
}

func BearingBetween(a, b orb.Point) float64 {
	return mathgeo.BearingBetween(a, b)
}

func BearingDiff(a, b float64) float64 {
	return mathgeo.BearingDiff(a, b)
}

func BearingAtMeasure(line orb.LineString, measure float64) float64 {
	return mathgeo.BearingAtMeasure(line, measure)
}

func SliceLineByMeasure(line orb.LineString, fromMeasure, toMeasure float64) (orb.LineString, error) {
	sliced, err := mathgeo.SliceLineByMeasure(line, fromMeasure, toMeasure)
	if err != nil {
		if errors.Is(err, mathgeo.ErrInvalidMeasureRange) {
			return nil, ErrInvalidMeasureRange
		}
		return nil, ErrInvalidLine
	}
	return sliced, nil
}

func PointAtMeasure(line orb.LineString, measure float64) (orb.Point, int) {
	return mathgeo.PointAtMeasure(line, measure)
}

type ProjectionCandidate struct {
	Point         orb.Point
	Measure       float64
	LineIndex     int
	DistanceMeter float64
}

func FindProjectionCandidates(line orb.LineString, point orb.Point) []ProjectionCandidate {
	raw := mathgeo.FindProjectionCandidates(line, point)
	out := make([]ProjectionCandidate, len(raw))
	for i, c := range raw {
		out[i] = ProjectionCandidate(c)
	}
	return out
}

func FindNearestProjectionAfterMeasure(line orb.LineString, point orb.Point, minMeasure float64) ProjectionCandidate {
	return ProjectionCandidate(mathgeo.FindNearestProjectionAfterMeasure(line, point, minMeasure))
}

func ProjectPointOnLine(line orb.LineString, point orb.Point) ProjectionCandidate {
	candidates := FindProjectionCandidates(line, point)
	if len(candidates) == 0 {
		return ProjectionCandidate{}
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.DistanceMeter < best.DistanceMeter {
			best = c
		}
	}
	return best
}
