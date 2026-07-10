package snaptoline

import "github.com/paulmach/orb"

const stopPassToleranceMeter = 5.0

// fillStopContext sets prev/curr/next stop fields on result.
//
// At halte (inside arrival radius): prev = halte before curr, curr = current halte, next = halte after curr.
// En route (left radius): prev = departed halte, curr = "", next = upcoming halte.
func fillStopContext(stops []Stop, line orb.LineString, point GPSPoint, snapped orb.Point, routeMeasure float64, cfg Config, activeSeg *Segment, result *SnapResult) {
	if result == nil || len(stops) == 0 {
		return
	}

	sorted := sortStopsByOrder(stops)
	if currStop := arrivedStop(sorted, point, snapped, cfg); currStop != nil {
		result.CurrStopID = currStop.ID
		result.CurrStopName = currStop.Name
		if prev := previousStopByOrder(sorted, currStop.Order); prev != nil {
			result.PrevStopID = prev.ID
			result.PrevStopName = prev.Name
		}
		if next := nextStopByOrder(sorted, currStop.Order); next != nil {
			result.NextStopID = next.ID
			result.NextStopName = next.Name
		}
		return
	}

	var nextStop *Stop
	if line != nil {
		nextStop = upcomingStopByMeasure(sorted, line, routeMeasure)
	} else if activeSeg != nil {
		nextStop = upcomingStopFromSegment(sorted, *activeSeg, routeMeasure)
	}
	if nextStop == nil {
		return
	}
	result.NextStopID = nextStop.ID
	result.NextStopName = nextStop.Name
	if prev := previousStopByOrder(sorted, nextStop.Order); prev != nil {
		result.PrevStopID = prev.ID
		result.PrevStopName = prev.Name
	}
}

func (s *Snapper) annotateStopContext(result *SnapResult, point GPSPoint) {
	if result == nil {
		return
	}
	var activeSeg *Segment
	for i := range s.segments {
		if s.segments[i].Order == result.SegmentOrder {
			activeSeg = &s.segments[i]
			break
		}
	}
	measure := s.RouteMeasure(result.SegmentOrder, result.Progress)
	fillStopContext(s.stops, s.line, point, result.SnappedPoint, measure, s.config, activeSeg, result)
}

func stopArrivalRadius(cfg Config) float64 {
	r := cfg.SegmentSwitchStopRadiusMeter
	if r <= 0 {
		r = DefaultRouteSegmentSwitchStopRadiusMeter
	}
	return r
}

func arrivedStop(stops []Stop, point GPSPoint, snapped orb.Point, cfg Config) *Stop {
	threshold := stopArrivalRadius(cfg)
	dwellSpeed := cfg.ClampDwellSpeedKmh
	if dwellSpeed <= 0 {
		dwellSpeed = DefaultRouteClampDwellSpeedKmh
	}
	isDwell := !cfg.UseSpeed || point.Speed <= dwellSpeed

	var best *Stop
	var bestDist float64
	for i := range stops {
		geoDist := DistanceMeter(point.Point, stops[i].Point)
		atStop := geoDist <= threshold
		if !atStop && isDwell {
			snapDist := DistanceMeter(snapped, stops[i].Point)
			if snapDist <= threshold {
				atStop = true
				geoDist = snapDist
			}
		}
		if !atStop {
			continue
		}
		if best == nil || geoDist < bestDist || (geoDist == bestDist && stops[i].Order > best.Order) {
			copy := stops[i]
			best = &copy
			bestDist = geoDist
		}
	}
	return best
}

func upcomingStopByMeasure(stops []Stop, line orb.LineString, routeMeasure float64) *Stop {
	for i := range stops {
		if stops[i].Order == 0 {
			continue
		}
		m := ProjectPointOnLine(line, stops[i].Point).Measure
		if m >= routeMeasure-stopPassToleranceMeter {
			copy := stops[i]
			return &copy
		}
	}
	return nil
}

func upcomingStopFromSegment(stops []Stop, seg Segment, routeMeasure float64) *Stop {
	if routeMeasure < seg.ToMeasure-stopPassToleranceMeter {
		copy := seg.ToStop
		return &copy
	}
	return nextStopByOrder(stops, seg.ToStop.Order)
}

func previousStopByOrder(stops []Stop, order int) *Stop {
	var best *Stop
	for i := range stops {
		if stops[i].Order >= order {
			continue
		}
		if best == nil || stops[i].Order > best.Order {
			copy := stops[i]
			best = &copy
		}
	}
	return best
}

func nextStopByOrder(stops []Stop, order int) *Stop {
	var best *Stop
	for i := range stops {
		if stops[i].Order <= order {
			continue
		}
		if best == nil || stops[i].Order < best.Order {
			copy := stops[i]
			best = &copy
		}
	}
	return best
}
