package snaptoline

import (
	"fmt"

	"github.com/paulmach/orb"
)

func BuildSegmentsFromProjectedStops(
	line orb.LineString,
	projectedStops []ProjectedStop,
	cfg Config,
) ([]Segment, error) {
	if len(projectedStops) < 2 {
		return nil, ErrInsufficientStops
	}

	segments := make([]Segment, 0, len(projectedStops)-1)
	totalLength := LineLengthMeter(line)

	for i := 0; i < len(projectedStops)-1; i++ {
		from := projectedStops[i]
		to := projectedStops[i+1]

		fromMeasure := from.Measure
		toMeasure := to.Measure

		if toMeasure <= fromMeasure {
			return nil, ErrInvalidMeasureRange
		}

		geometry, err := SliceLineByMeasure(line, fromMeasure, toMeasure)
		if err != nil {
			return nil, err
		}

		bearing := BearingAtMeasure(geometry, 0)
		if len(geometry) >= 2 {
			bearing = BearingBetween(geometry[0], geometry[len(geometry)-1])
		}

		direction := resolveSegmentDirection(cfg, to.IsLoopClosure, i)

		seg := Segment{
			ID:            segmentID(from.Stop, to.Stop, i+1),
			FromStop:      from.Stop,
			ToStop:        to.Stop,
			FromMeasure:   fromMeasure,
			ToMeasure:     toMeasure,
			Geometry:      geometry,
			Order:         i + 1,
			Direction:     direction,
			Bearing:       bearing,
			IsLooping:     cfg.Looping,
			IsLoopClosing: to.IsLoopClosure,
		}

		segments = append(segments, seg)
	}

	if cfg.Looping && len(segments) > 0 {
		last := segments[len(segments)-1]
		if last.ToMeasure < totalLength*0.95 && projectedStops[len(projectedStops)-1].IsLoopClosure {
			return nil, fmt.Errorf("snaptoline: loop closing segment must reach end of line")
		}
	}

	return segments, nil
}

func resolveSegmentDirection(cfg Config, isLoopClosure bool, segmentIndex int) DirectionType {
	if segmentIndex >= 0 && segmentIndex < len(cfg.SegmentDirections) {
		if cfg.SegmentDirections[segmentIndex] != "" {
			return cfg.SegmentDirections[segmentIndex]
		}
	}
	if cfg.Looping {
		return DirectionLoop
	}
	if cfg.UseTripDirection && cfg.TripDirection != DirectionUnknown {
		return cfg.TripDirection
	}
	return DirectionUnknown
}

func BuildSegments(
	line orb.LineString,
	stops []Stop,
	cfg Config,
) ([]Segment, error) {
	projected, err := ProjectStopsSequential(line, stops, cfg)
	if err != nil {
		return nil, err
	}
	return BuildSegmentsFromProjectedStops(line, projected, cfg)
}

func nearestStopID(stops []Stop, point orb.Point) string {
	if len(stops) == 0 {
		return ""
	}

	best := stops[0]
	bestDist := DistanceMeter(point, best.Point)

	for _, stop := range stops[1:] {
		d := DistanceMeter(point, stop.Point)
		if d < bestDist {
			best = stop
			bestDist = d
		}
	}

	return best.ID
}

func segmentProgress(seg Segment, measure float64) float64 {
	length := seg.ToMeasure - seg.FromMeasure
	if length <= 0 {
		return 0
	}
	return (measure - seg.FromMeasure) / length
}
