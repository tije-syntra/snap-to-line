package snaptoline

import "github.com/paulmach/orb"

const (
	minSnapContinuityGeoJumpM = 8.0
)

func foldedMeasureSpread(viable []ProjectionCandidate) float64 {
	if len(viable) < 2 {
		return 0
	}
	minM, maxM := viable[0].Measure, viable[0].Measure
	for _, c := range viable[1:] {
		if c.Measure < minM {
			minM = c.Measure
		}
		if c.Measure > maxM {
			maxM = c.Measure
		}
	}
	return maxM - minM
}

func foldedSegmentMeasureSpread(cfg Config) float64 {
	if cfg.FoldedSegmentMeasureSpreadM > 0 {
		return cfg.FoldedSegmentMeasureSpreadM
	}
	return DefaultFoldedSegmentMeasureSpreadM
}

// isAmbiguousSegmentGeometry is true when overlapping geometry yields multiple distant projections.
func isAmbiguousSegmentGeometry(viable []ProjectionCandidate, cfg Config) bool {
	if len(viable) < 2 {
		return false
	}
	if isFoldedSegment(viable, cfg) {
		return true
	}
	return foldedMeasureSpread(viable) > foldedSegmentMeasureSpread(cfg)
}

func snapContinuityEnabled(cfg Config) bool {
	return cfg.SnapContinuityFromPrevious
}

func maxAllowedSnapJumpM(cfg Config, gpsMovement float64) float64 {
	jumpSlack := cfg.SnappedJumpSlackMeter
	if jumpSlack <= 0 {
		jumpSlack = DefaultRouteSnappedJumpSlackMeter
	}

	maxJump := gpsMovement*1.5 + jumpSlack
	if maxJump < minSnapContinuityGeoJumpM {
		maxJump = minSnapContinuityGeoJumpM
	}

	maxFwd := cfg.MaxForwardSnapMeter
	if maxFwd > 0 && maxJump > maxFwd {
		maxJump = maxFwd
	}
	return maxJump
}

func maxAllowedMeasureAdvanceM(cfg Config, gpsMovement float64) float64 {
	maxFwd := cfg.MaxForwardSnapMeter
	if maxFwd <= 0 {
		advanceSlack := cfg.MeasureAdvanceSlackMeter
		if advanceSlack <= 0 {
			advanceSlack = DefaultRouteMeasureAdvanceSlackMeter
		}
		if gpsMovement < 1 {
			gpsMovement = 1
		}
		return gpsMovement*3 + advanceSlack
	}
	_ = gpsMovement
	return maxFwd
}

func snapContinuityScore(c ProjectionCandidate, lastSnapped orb.Point, hasLast bool) float64 {
	if !hasLast || !hasLastSnapped(lastSnapped) {
		return c.DistanceMeter
	}
	return DistanceMeter(lastSnapped, c.Point)*2 + c.DistanceMeter
}

func pickBestContinuityAmongViable(viable []ProjectionCandidate, lastSnapped orb.Point, hasLast bool) ProjectionCandidate {
	if len(viable) == 0 {
		return ProjectionCandidate{}
	}
	best := viable[0]
	bestScore := snapContinuityScore(best, lastSnapped, hasLast)
	for _, c := range viable[1:] {
		score := snapContinuityScore(c, lastSnapped, hasLast)
		if score < bestScore {
			best = c
			bestScore = score
		}
	}
	return best
}

// enforceSnapContinuityFromPrevious limits snap jumps relative to the previous snapped position.
// GPS error alone must not move the bus far along folded or overlapping geometry.
func (s *Snapper) enforceSnapContinuityFromPrevious(best *Candidate, point GPSPoint) *Candidate {
	if !snapContinuityEnabled(s.config) || best == nil || s.state.LastBest == nil {
		return best
	}

	ref := s.state.LastBest
	loopWrap := isLoopWrapTransition(ref.Segment.Order, best.Segment.Order, len(s.segments), s.config.Looping)
	if loopWrap {
		return best
	}

	gpsMovement := 0.0
	if s.state.LastPoint != nil {
		gpsMovement = DistanceMeter(s.state.LastPoint.Point, point.Point)
	}

	snapJump := DistanceMeter(ref.SnappedPoint, best.SnappedPoint)
	maxGeoJump := maxAllowedSnapJumpM(s.config, gpsMovement)
	measureAdv := best.Measure - ref.Measure
	maxMeasureAdv := maxAllowedMeasureAdvanceM(s.config, gpsMovement)

	if snapJump <= maxGeoJump && measureAdv <= maxMeasureAdv+backwardMeasureEpsilonM {
		return best
	}

	targetM := best.Measure
	if targetM > ref.Measure+maxMeasureAdv {
		targetM = ref.Measure + maxMeasureAdv
	}
	if targetM < ref.Measure-backwardMeasureEpsilonM {
		targetM = ref.Measure
	}
	if seg := s.segmentAtRouteMeasure(targetM); seg != nil {
		if result, cand := s.candidateAtRouteMeasure(*seg, targetM, point); cand != nil {
			c := *cand
			contJump := DistanceMeter(ref.SnappedPoint, c.SnappedPoint)
			if contJump <= maxGeoJump+1 || c.Measure <= ref.Measure+maxMeasureAdv+backwardMeasureEpsilonM {
				_ = result
				return &c
			}
		}
	}

	continued := s.candidateOnSegment(ref.Segment, point)
	tol := s.config.MeasureRegressionToleranceMeter
	if tol <= 0 {
		tol = DefaultRouteMeasureRegressionToleranceMeter
	}
	contJump := DistanceMeter(ref.SnappedPoint, continued.SnappedPoint)
	if continued.Measure >= ref.Measure-tol &&
		contJump <= maxGeoJump+2 &&
		continued.Measure <= ref.Measure+maxMeasureAdv+backwardMeasureEpsilonM {
		return &continued
	}

	if s.shouldCreepForwardFromPrevious(point, ref) {
		if _, c := s.creepForwardOnSegment(point, ref); c != nil && c.Measure > ref.Measure+backwardMeasureEpsilonM {
			return c
		}
	}

	frozen := *ref
	frozen.DistanceMeter = DistanceMeter(point.Point, frozen.SnappedPoint)
	return &frozen
}

func (s *Snapper) applySnapContinuityResult(best *Candidate, point GPSPoint, result *SnapResult) (*SnapResult, *Candidate) {
	if result != nil && result.HeldReason == "forward_cap" {
		return nil, best
	}
	adjusted := s.enforceSnapContinuityFromPrevious(best, point)
	if adjusted == nil || adjusted == best {
		return nil, best
	}
	contResult := s.resultFromCandidate(*adjusted, point)
	if s.state.LastBest != nil &&
		DistanceMeter(adjusted.SnappedPoint, s.state.LastBest.SnappedPoint) < 0.5 {
		contResult.HeldSegment = true
	}
	if contResult.HeldReason == "" {
		contResult.HeldReason = "snap_continuity"
	}
	minConf := s.holdLastSegmentMinConfidence()
	if contResult.Confidence < minConf {
		contResult.Confidence = minConf
	}
	contResult.IsOffRoute = contResult.DistanceMeter > s.config.MaxSnapDistanceMeter
	return contResult, adjusted
}

// ensureForwardCreepWhenStuck advances along the route when GPS moves forward but the snap
// stays at the previous position (common when raw GPS drifts off-route on folded geometry).
func (s *Snapper) ensureForwardCreepWhenStuck(best *Candidate, point GPSPoint) (*SnapResult, *Candidate) {
	if !snapContinuityEnabled(s.config) || s.state.LastBest == nil || best == nil || s.state.LastPoint == nil {
		return nil, nil
	}

	ref := s.state.LastBest
	if best.Segment.Order != ref.Segment.Order {
		return nil, nil
	}

	measureAdv := best.Measure - ref.Measure
	snapJump := DistanceMeter(ref.SnappedPoint, best.SnappedPoint)
	if measureAdv > backwardMeasureEpsilonM || snapJump > backwardMeasureEpsilonM {
		return nil, nil
	}

	gpsMovement := DistanceMeter(s.state.LastPoint.Point, point.Point)
	minMov := s.config.MinMovementMeter
	if minMov <= 0 {
		minMov = 3
	}
	if gpsMovement < minMov && !s.shouldCreepForwardFromPrevious(point, ref) {
		return nil, nil
	}
	if !s.shouldCreepForwardFromPrevious(point, ref) {
		return nil, nil
	}

	crept, c := s.creepForwardOnSegment(point, ref)
	if crept == nil || c == nil || c.Measure <= ref.Measure+backwardMeasureEpsilonM {
		return nil, nil
	}
	if c.Segment.Order > ref.Segment.Order {
		crept, c = s.candidateAtRouteMeasure(ref.Segment, ref.Segment.ToMeasure, point)
		if crept == nil || c == nil || c.Measure <= ref.Measure+backwardMeasureEpsilonM {
			return nil, nil
		}
	}

	result := s.resultFromCandidate(*c, point)
	result.HeldSegment = true
	result.HeldReason = "snap_continuity_creep"
	minConf := s.holdLastSegmentMinConfidence()
	if result.Confidence < minConf {
		result.Confidence = minConf
	}
	result.IsOffRoute = result.DistanceMeter > s.config.MaxSnapDistanceMeter
	return result, c
}
