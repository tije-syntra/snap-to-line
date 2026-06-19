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

// ProjectPointOnLineContinued prefers projections near prevRelMeasure when multiple
// viable projections exist on folded/overlapping geometry.
func ProjectPointOnLineContinued(
	line orb.LineString,
	point orb.Point,
	prevRelMeasure float64,
	prevPoint *GPSPoint,
	lastSnapped orb.Point,
	cfg Config,
) ProjectionCandidate {
	if prevRelMeasure <= 0 {
		return ProjectPointOnLine(line, point)
	}

	candidates := FindProjectionCandidates(line, point)
	viable := make([]ProjectionCandidate, 0, len(candidates))
	for _, c := range candidates {
		if c.DistanceMeter <= cfg.MaxSnapDistanceMeter {
			viable = append(viable, c)
		}
	}
	if len(viable) == 0 {
		return ProjectPointOnLine(line, point)
	}
	if len(viable) == 1 {
		return viable[0]
	}

	tol := cfg.MeasureRegressionToleranceMeter
	if tol <= 0 {
		tol = 30
	}

	best := viable[0]
	for _, c := range viable[1:] {
		dBest := measureDelta(best.Measure, prevRelMeasure)
		dCand := measureDelta(c.Measure, prevRelMeasure)

		switch {
		case dCand < dBest:
			best = c
		case dCand == dBest && c.DistanceMeter < best.DistanceMeter:
			best = c
		}
	}

	nearest := ProjectPointOnLine(line, point)
	if nearest.Measure < prevRelMeasure-tol && measureDelta(best.Measure, prevRelMeasure)+5 < measureDelta(nearest.Measure, prevRelMeasure) {
		return best
	}

	jumpSlack := cfg.SnappedJumpSlackMeter
	if jumpSlack <= 0 {
		jumpSlack = DefaultRouteSnappedJumpSlackMeter
	}
	if hasLastSnapped(lastSnapped) && prevPoint != nil {
		movement := DistanceMeter(prevPoint.Point, point)
		if movement < 1 {
			movement = 1
		}
		jumpNearest := DistanceMeter(lastSnapped, nearest.Point)
		jumpBest := DistanceMeter(lastSnapped, best.Point)
		if jumpNearest > movement*0.75+jumpSlack && jumpBest < jumpNearest {
			return best
		}
	}

	if measureDelta(best.Measure, prevRelMeasure)+1 < measureDelta(nearest.Measure, prevRelMeasure) {
		return best
	}
	return nearest
}

func measureDelta(a, b float64) float64 {
	d := a - b
	if d < 0 {
		return -d
	}
	return d
}

func hasLastSnapped(p orb.Point) bool {
	return p[0] != 0 || p[1] != 0
}
