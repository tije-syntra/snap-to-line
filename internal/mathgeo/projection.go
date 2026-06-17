package mathgeo

import (
	"github.com/paulmach/orb"
)

type ProjectionCandidate struct {
	Point         orb.Point
	Measure       float64
	LineIndex     int
	DistanceMeter float64
}

func ProjectPointOnSegment(a, b, p orb.Point) (orb.Point, float64) {
	ax, ay := a[0], a[1]
	bx, by := b[0], b[1]
	px, py := p[0], p[1]

	dx := bx - ax
	dy := by - ay

	if dx == 0 && dy == 0 {
		return a, 0
	}

	t := ((px-ax)*dx + (py-ay)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	projected := orb.Point{ax + t*dx, ay + t*dy}
	return projected, t
}

func FindProjectionCandidates(line orb.LineString, point orb.Point) []ProjectionCandidate {
	if len(line) < 2 {
		return nil
	}

	cumulative := 0.0
	candidates := make([]ProjectionCandidate, 0, len(line)-1)

	for i := 0; i < len(line)-1; i++ {
		a := line[i]
		b := line[i+1]
		segLen := DistanceMeter(a, b)

		projected, t := ProjectPointOnSegment(a, b, point)
		measure := cumulative + segLen*t

		candidates = append(candidates, ProjectionCandidate{
			Point:         projected,
			Measure:       measure,
			LineIndex:     i,
			DistanceMeter: DistanceMeter(point, projected),
		})

		cumulative += segLen
	}

	return candidates
}

func FindNearestProjectionAfterMeasure(line orb.LineString, point orb.Point, minMeasure float64) ProjectionCandidate {
	candidates := FindProjectionCandidates(line, point)

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

	if best != nil {
		return *best
	}

	// Fallback: use the last point on the line.
	total := LineLengthMeter(line)
	last := line[len(line)-1]
	return ProjectionCandidate{
		Point:         last,
		Measure:       total,
		LineIndex:     len(line) - 2,
		DistanceMeter: DistanceMeter(point, last),
	}
}

func FindProjectionNearMeasure(line orb.LineString, point orb.Point, targetMeasure float64, searchWindow float64) ProjectionCandidate {
	candidates := FindProjectionCandidates(line, point)

	var best *ProjectionCandidate
	for i := range candidates {
		c := &candidates[i]
		delta := c.Measure - targetMeasure
		if delta < 0 {
			delta = -delta
		}
		if searchWindow > 0 && delta > searchWindow {
			continue
		}
		if best == nil || c.DistanceMeter < best.DistanceMeter {
			best = c
		}
	}

	if best != nil {
		return *best
	}

	return FindNearestProjectionAfterMeasure(line, point, 0)
}
