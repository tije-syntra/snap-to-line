package snaptoline

import (
	"math"
	"time"
)

func (s *Snapper) shouldClampBackward(result *SnapResult) bool {
	if !s.config.PreventBackwardTransition || s.state.LastBest == nil {
		return false
	}
	if isLoopWrapTransition(s.state.LastBest.Segment.Order, result.SegmentOrder, len(s.segments), s.config.Looping) {
		return false
	}
	// Never accept a lower segment order on live routes (regardless of confidence/speed).
	return result.SegmentOrder < s.state.LastBest.Segment.Order
}

func (s *Snapper) shouldClampMeasureRegression(result *SnapResult) bool {
	if s.state.LastBest == nil {
		return false
	}
	tol := s.config.MeasureRegressionToleranceMeter
	if tol <= 0 {
		return false
	}
	if isLoopWrapTransition(s.state.LastBest.Segment.Order, result.SegmentOrder, len(s.segments), s.config.Looping) {
		return false
	}
	resMeasure := s.RouteMeasure(result.SegmentOrder, result.Progress)
	return resMeasure < s.state.LastBest.Measure-tol
}

func (s *Snapper) shouldClampLateral(result *SnapResult, point GPSPoint) bool {
	if s.state.LastBest == nil || s.state.LastPoint == nil {
		return false
	}
	if s.config.ClampBackwardMinConfidence <= 0 {
		return false
	}

	jump := DistanceMeter(s.state.LastBest.SnappedPoint, result.SnappedPoint)
	movement := DistanceMeter(s.state.LastPoint.Point, point.Point)
	if movement < 1 {
		movement = 1
	}

	jumpSlack := s.config.SnappedJumpSlackMeter
	if jumpSlack <= 0 {
		jumpSlack = DefaultRouteSnappedJumpSlackMeter
	}

	dwellSpeed := s.config.ClampDwellSpeedKmh
	if dwellSpeed <= 0 {
		dwellSpeed = DefaultRouteClampDwellSpeedKmh
	}
	isDwell := point.Speed < 3 || (s.config.UseSpeed && point.Speed <= dwellSpeed) || movement < s.config.MinMovementMeter

	confLimit := 0.70
	if isDwell {
		confLimit = 0.85
	}
	if result.Confidence >= confLimit {
		return false
	}

	if isDwell && jump > jumpSlack {
		return true
	}
	return jump > movement*0.75+jumpSlack && jump > 4
}

func (s *Snapper) holdReferenceCandidate() *Candidate {
	if s.state.LastBest != nil && s.state.LastBest.Segment.Order > 0 {
		return s.state.LastBest
	}
	if s.state.LastGood != nil && s.state.LastGood.Segment.Order > 0 {
		return s.state.LastGood
	}
	return nil
}

func (s *Snapper) clampToPreviousSegment(point GPSPoint) *SnapResult {
	result, _ := s.clampToPreviousSegmentWithMaxDist(point, s.config.MaxSnapDistanceMeter)
	return result
}

func (s *Snapper) clampToPreviousSegmentWithMaxDist(point GPSPoint, maxSnapDist float64) (*SnapResult, *Candidate) {
	ref := s.holdReferenceCandidate()
	if ref == nil {
		return nil, nil
	}
	seg := ref.Segment
	prevRel := ref.Measure - seg.FromMeasure
	lastAbs := ref.Measure
	lastSnapped := ref.SnappedPoint

	tol := s.config.MeasureRegressionToleranceMeter
	if tol <= 0 {
		tol = DefaultRouteMeasureRegressionToleranceMeter
	}
	advanceSlack := s.config.MeasureAdvanceSlackMeter
	if advanceSlack <= 0 {
		advanceSlack = DefaultRouteMeasureAdvanceSlackMeter
	}

	movement := 0.0
	if s.state.LastPoint != nil {
		movement = DistanceMeter(s.state.LastPoint.Point, point.Point)
	}
	maxAdvance := movement*1.5 + advanceSlack
	if movement > 0 && movement < 1 {
		maxAdvance = 1.5 + advanceSlack
	}

	proj := ProjectPointOnLineContinued(seg.Geometry, point.Point, prevRel, s.state.LastPoint, lastSnapped, s.config)

	minRel := prevRel - tol
	maxRel := prevRel + maxAdvance
	if forward, ok := ForwardProjectionOnSegment(seg.Geometry, point.Point, minRel, maxRel, maxSnapDist); ok {
		absForward := seg.FromMeasure + forward.Measure
		absProj := seg.FromMeasure + proj.Measure
		if absForward >= lastAbs-tol && (absProj < lastAbs-tol || absForward > absProj) {
			proj = forward
		}
	}

	if seg.FromMeasure+proj.Measure < lastAbs-tol {
		after := FindNearestProjectionAfterMeasure(seg.Geometry, point.Point, prevRel)
		if after.DistanceMeter <= maxSnapDist && seg.FromMeasure+after.Measure >= lastAbs-tol {
			proj = after
		}
	}

	if seg.FromMeasure+proj.Measure < lastAbs-tol {
		proj = ProjectionCandidate{
			Point:         lastSnapped,
			Measure:       prevRel,
			LineIndex:     0,
			DistanceMeter: DistanceMeter(point.Point, lastSnapped),
		}
	}

	candidate := s.candidateFromProjection(seg, point, proj)
	result := s.resultFromCandidate(candidate, point)
	return result, &candidate
}

func (s *Snapper) holdLastSegmentMaxDistM() float64 {
	if s.config.HoldLastSegmentMaxDistM > 0 {
		return s.config.HoldLastSegmentMaxDistM
	}
	if s.config.MaxSnapDistanceMeter > 0 {
		return math.Max(DefaultRouteHoldLastSegmentMaxDistM, s.config.MaxSnapDistanceMeter*2)
	}
	return DefaultRouteHoldLastSegmentMaxDistM
}

func (s *Snapper) holdLastSegmentMaxAgeMs() int64 {
	if s.config.HoldLastSegmentMaxAgeMs > 0 {
		return s.config.HoldLastSegmentMaxAgeMs
	}
	return DefaultRouteHoldLastSegmentMaxAgeMs
}

func (s *Snapper) holdLastSegmentMinConfidence() float64 {
	if s.config.HoldLastSegmentMinConfidence > 0 {
		return s.config.HoldLastSegmentMinConfidence
	}
	return DefaultRouteHoldLastSegmentMinConfidence
}

func (s *Snapper) holdLastSegmentAgeOK(point GPSPoint) bool {
	maxAge := s.holdLastSegmentMaxAgeMs()
	if s.state.LastTimestamp <= 0 {
		return true
	}
	now := point.Timestamp
	if now <= 0 {
		now = time.Now().UnixMilli()
	}
	delta := now - s.state.LastTimestamp
	if delta < 0 {
		return true
	}
	return delta <= maxAge
}

func (s *Snapper) applyHeldSegmentResult(result *SnapResult, reason string) *SnapResult {
	minConf := s.holdLastSegmentMinConfidence()
	conf := result.Confidence * 0.5
	if conf < minConf {
		conf = minConf
	}
	result.Confidence = clampConfidence(conf)
	result.HeldSegment = true
	result.HeldReason = reason
	// Keep route context for ETA; lateral distance remains in DistanceMeter.
	result.IsOffRoute = false
	result.RejectedReason = ""
	return result
}

// holdLastSegmentOnMiss projects onto the previous segment when findCandidates returns empty.
func (s *Snapper) holdLastSegmentOnMiss(point GPSPoint) (*SnapResult, *Candidate) {
	if !s.config.HoldLastSegmentOnMiss || s.holdReferenceCandidate() == nil {
		return nil, nil
	}
	if !s.holdLastSegmentAgeOK(point) {
		return nil, nil
	}

	// Use a generous projection window so hold works beyond normal max snap distance.
	projMax := s.holdLastSegmentMaxDistM()
	if projMax < 500 {
		projMax = 500
	}
	result, candidate := s.clampToPreviousSegmentWithMaxDist(point, projMax)
	if result == nil || candidate == nil {
		return nil, nil
	}
	if result.SegmentID == "" || result.SegmentOrder <= 0 {
		return nil, nil
	}

	result = s.applyHeldSegmentResult(result, "no_candidates")
	return result, candidate
}

func (s *Snapper) tryHoldLastSegment(point GPSPoint, reason string) (*SnapResult, *Candidate) {
	if !s.config.HoldLastSegmentOnMiss {
		return nil, nil
	}
	if result, best := s.holdLastSegmentOnMiss(point); result != nil {
		if reason != "" && reason != "no_candidates" {
			result.HeldReason = reason
		}
		return result, best
	}
	return nil, nil
}

func (s *Snapper) stabilizeSameSegmentCandidate(best *Candidate, point GPSPoint) *Candidate {
	if s.state.LastBest == nil || best == nil || best.Segment.Order != s.state.LastBest.Segment.Order {
		return best
	}

	jumpSlack := s.config.SnappedJumpSlackMeter
	if jumpSlack <= 0 {
		jumpSlack = DefaultRouteSnappedJumpSlackMeter
	}
	movement := 1.0
	if s.state.LastPoint != nil {
		movement = DistanceMeter(s.state.LastPoint.Point, point.Point)
		if movement < 1 {
			movement = 1
		}
	}

	jump := DistanceMeter(s.state.LastBest.SnappedPoint, best.SnappedPoint)
	maxJump := movement*1.25 + jumpSlack
	if jump <= maxJump || jump <= 5 {
		return best
	}

	continued := s.candidateOnSegment(s.state.LastBest.Segment, point)
	tol := s.config.MeasureRegressionToleranceMeter
	if tol <= 0 {
		tol = DefaultRouteMeasureRegressionToleranceMeter
	}
	if continued.Measure >= s.state.LastBest.Measure-tol {
		return &continued
	}
	return best
}

func (s *Snapper) softMaxSnapDist() float64 {
	if s.config.MaxSnapDistanceMeter <= 0 {
		return 18
	}
	soft := s.config.MaxSnapDistanceMeter * 0.65
	if soft < 12 {
		return 12
	}
	return soft
}

// preferNearbyOnActiveSegment keeps snap on the current segment when it is closer to raw GPS.
func (s *Snapper) preferNearbyOnActiveSegment(best *Candidate, point GPSPoint) *Candidate {
	if s.state.LastBest == nil || best == nil {
		return best
	}

	if forwardDepartLatchActive(s.state, s.state.LastBest.Segment.Order) {
		return best
	}

	soft := s.softMaxSnapDist()
	if best.DistanceMeter <= soft {
		return best
	}

	lastSeg := s.state.LastBest.Segment
	onLast := s.candidateOnSegment(lastSeg, point)
	if onLast.DistanceMeter > s.config.MaxSnapDistanceMeter {
		return best
	}

	tol := s.config.MeasureRegressionToleranceMeter
	if tol <= 0 {
		tol = DefaultRouteMeasureRegressionToleranceMeter
	}
	measureOK := onLast.Measure >= s.state.LastBest.Measure-tol

	if best.Segment.Order == lastSeg.Order && onLast.DistanceMeter < best.DistanceMeter {
		return &onLast
	}

	if onLast.DistanceMeter+2 < best.DistanceMeter && measureOK {
		return &onLast
	}

	if best.Segment.Order != lastSeg.Order && best.DistanceMeter > soft && onLast.DistanceMeter <= soft && measureOK {
		return &onLast
	}

	return best
}

func (s *Snapper) candidateOnSegment(seg Segment, point GPSPoint) Candidate {
	proj := projectOntoSegment(seg, point, s.state.LastPoint, s.state, s.config)
	return s.candidateFromProjection(seg, point, proj)
}

func (s *Snapper) candidateFromProjection(seg Segment, point GPSPoint, proj ProjectionCandidate) Candidate {
	absMeasure := seg.FromMeasure + proj.Measure
	lineBearing := BearingAtMeasure(seg.Geometry, proj.Measure)

	busBearing, hasBearing := resolveBusBearing(point, s.state.LastPoint, s.config)
	weaken := shouldWeakenDirectionValidation(point, s.state.LastPoint, s.config, s.state.TurnaroundValidated)
	emission := EmissionScore(proj.DistanceMeter, s.config.MaxSnapDistanceMeter)
	dirScore, _ := scoreDirection(busBearing, hasBearing, lineBearing, s.config, weaken)
	tripScore := TripDirectionScore(seg.Direction, s.config.TripDirection)

	return Candidate{
		Segment:            seg,
		Measure:            absMeasure,
		SnappedPoint:       proj.Point,
		DistanceMeter:      proj.DistanceMeter,
		LineBearing:        lineBearing,
		EmissionScore:      emission,
		DirectionScore:     dirScore,
		TripDirectionScore: tripScore,
	}
}

func (s *Snapper) resultFromCandidate(best Candidate, point GPSPoint) *SnapResult {
	busBearing, hasBearing := resolveBusBearing(point, s.state.LastPoint, s.config)
	weaken := shouldWeakenDirectionValidation(point, s.state.LastPoint, s.config, s.state.TurnaroundValidated)
	_, directionDiff := scoreDirection(busBearing, hasBearing, best.LineBearing, s.config, weaken)

	if !hasBearing && s.state.LastBest != nil {
		busBearing = s.state.LastBest.LineBearing
	}

	return &SnapResult{
		OriginalPoint: point.Point,
		SnappedPoint:  best.SnappedPoint,
		SegmentID:     best.Segment.ID,
		SegmentOrder:  best.Segment.Order,
		Direction:     best.Segment.Direction,
		NearestStopID: nearestStopID(s.stops, best.SnappedPoint),
		DistanceMeter: best.DistanceMeter,
		Progress:      segmentProgress(best.Segment, best.Measure),
		BusBearing:    busBearing,
		LineBearing:   best.LineBearing,
		DirectionDiff: directionDiff,
		Confidence:    clampConfidence(confidenceFromScores(best)),
		IsOffRoute:    best.DistanceMeter > s.config.MaxSnapDistanceMeter,
	}
}
