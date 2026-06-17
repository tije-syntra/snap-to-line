package mathgeo

import (
	"errors"

	"github.com/paulmach/orb"
)

var (
	ErrInvalidLine         = errors.New("mathgeo: line must contain at least two points")
	ErrInvalidMeasureRange = errors.New("mathgeo: invalid measure range")
)

func LineLengthMeter(line orb.LineString) float64 {
	if len(line) < 2 {
		return 0
	}

	total := 0.0
	for i := 0; i < len(line)-1; i++ {
		total += DistanceMeter(line[i], line[i+1])
	}
	return total
}

func PointAtMeasure(line orb.LineString, measure float64) (orb.Point, int) {
	if len(line) == 0 {
		return orb.Point{}, 0
	}
	if len(line) == 1 || measure <= 0 {
		return line[0], 0
	}

	total := LineLengthMeter(line)
	if measure >= total {
		return line[len(line)-1], len(line) - 2
	}

	cumulative := 0.0
	for i := 0; i < len(line)-1; i++ {
		a := line[i]
		b := line[i+1]
		segLen := DistanceMeter(a, b)
		if cumulative+segLen >= measure {
			t := (measure - cumulative) / segLen
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			return orb.Point{
				a[0] + t*(b[0]-a[0]),
				a[1] + t*(b[1]-a[1]),
			}, i
		}
		cumulative += segLen
	}

	return line[len(line)-1], len(line) - 2
}

func BearingAtMeasure(line orb.LineString, measure float64) float64 {
	if len(line) < 2 {
		return 0
	}

	total := LineLengthMeter(line)
	if measure >= total {
		a := line[len(line)-2]
		b := line[len(line)-1]
		return BearingBetween(a, b)
	}

	cumulative := 0.0
	for i := 0; i < len(line)-1; i++ {
		a := line[i]
		b := line[i+1]
		segLen := DistanceMeter(a, b)
		if cumulative+segLen >= measure || i == len(line)-2 {
			return BearingBetween(a, b)
		}
		cumulative += segLen
	}

	a := line[len(line)-2]
	b := line[len(line)-1]
	return BearingBetween(a, b)
}

func SliceLineByMeasure(line orb.LineString, fromMeasure, toMeasure float64) (orb.LineString, error) {
	if len(line) < 2 {
		return nil, ErrInvalidLine
	}
	if toMeasure <= fromMeasure {
		return nil, ErrInvalidMeasureRange
	}

	startPoint, startIdx := PointAtMeasure(line, fromMeasure)
	endPoint, endIdx := PointAtMeasure(line, toMeasure)

	result := orb.LineString{startPoint}

	for i := startIdx + 1; i <= endIdx && i < len(line); i++ {
		result = append(result, line[i])
	}

	last := result[len(result)-1]
	if last[0] != endPoint[0] || last[1] != endPoint[1] {
		result = append(result, endPoint)
	}

	if len(result) < 2 {
		return nil, ErrInvalidMeasureRange
	}

	return result, nil
}
