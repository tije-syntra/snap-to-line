package snaptoline

import (
	"math"

	"github.com/paulmach/orb"
)

const (
	branchOscillationMeasureDeltaM    = 15.0
	branchDuplicateMeasureToleranceM  = 3.0
	branchDuplicateDistanceToleranceM = 2.0
)

func foldedSegmentBranchLockEnabled(cfg Config) bool {
	return cfg.FoldedSegmentBranchLock
}

func foldedSegmentMinViable(cfg Config) int {
	if cfg.FoldedSegmentMinViable > 0 {
		return cfg.FoldedSegmentMinViable
	}
	return DefaultFoldedSegmentMinViable
}

func branchLockSearchWindow(cfg Config) float64 {
	if cfg.BranchLockSearchWindowM > 0 {
		return cfg.BranchLockSearchWindowM
	}
	return DefaultBranchLockSearchWindowM
}

func branchUnlockNormalTicks(cfg Config) int {
	if cfg.BranchUnlockNormalTicks > 0 {
		return cfg.BranchUnlockNormalTicks
	}
	return DefaultBranchUnlockNormalTicks
}

func isFoldedSegment(viable []ProjectionCandidate, cfg Config) bool {
	return len(viable) >= foldedSegmentMinViable(cfg)
}

func viableProjections(line orb.LineString, point orb.Point, maxDist float64) []ProjectionCandidate {
	all := FindProjectionCandidates(line, point)
	out := make([]ProjectionCandidate, 0, len(all))
	for _, c := range all {
		if c.DistanceMeter > maxDist {
			continue
		}
		if isDuplicateProjection(out, c) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func isDuplicateProjection(existing []ProjectionCandidate, c ProjectionCandidate) bool {
	for _, e := range existing {
		if measureDelta(e.Measure, c.Measure) < branchDuplicateMeasureToleranceM &&
			DistanceMeter(e.Point, c.Point) < branchDuplicateDistanceToleranceM {
			return true
		}
	}
	return false
}

func pickNearestProjection(viable []ProjectionCandidate) ProjectionCandidate {
	best := viable[0]
	for _, c := range viable[1:] {
		if c.DistanceMeter < best.DistanceMeter {
			best = c
		}
	}
	return best
}

func projectionLineIndex(line orb.LineString, point orb.Point, relMeasure float64) int {
	cands := FindProjectionCandidates(line, point)
	bestIdx := -1
	bestDelta := math.MaxFloat64
	for _, c := range cands {
		d := measureDelta(c.Measure, relMeasure)
		if d < bestDelta {
			bestDelta = d
			bestIdx = c.LineIndex
		}
	}
	return bestIdx
}

// projectOntoSegment picks a projection for one segment, applying branch lock on folded geometry.
func projectOntoSegment(
	seg Segment,
	point GPSPoint,
	prev *GPSPoint,
	state *ViterbiState,
	cfg Config,
) ProjectionCandidate {
	prevRel := 0.0
	var lastSnapped orb.Point
	hasLast := state != nil && state.LastBest != nil
	if hasLast {
		lastSnapped = state.LastBest.SnappedPoint
		if state.LastBest.Segment.Order == seg.Order {
			prevRel = state.LastBest.Measure - seg.FromMeasure
		}
	}

	if foldedSegmentBranchLockEnabled(cfg) && state != nil && state.BranchLock != nil &&
		state.BranchLock.SegmentOrder == seg.Order {
		return projectOnLockedBranch(seg, point.Point, state.BranchLock, cfg)
	}

	viable := viableProjections(seg.Geometry, point.Point, cfg.MaxSnapDistanceMeter)
	if foldedSegmentBranchLockEnabled(cfg) && isAmbiguousSegmentGeometry(viable, cfg) {
		return pickBestContinuityAmongViable(viable, lastSnapped, hasLast)
	}

	return ProjectPointOnLineContinued(seg.Geometry, point.Point, prevRel, prev, lastSnapped, cfg)
}

func projectOnLockedBranch(seg Segment, point orb.Point, lock *BranchLock, cfg Config) ProjectionCandidate {
	window := branchLockSearchWindow(cfg)
	target := lock.LockedRelMeasure
	minRel := target - window
	if minRel < 0 {
		minRel = 0
	}
	maxRel := target + window
	segLen := seg.ToMeasure - seg.FromMeasure
	if segLen > 0 && maxRel > segLen {
		maxRel = segLen
	}

	if proj, ok := ForwardProjectionOnSegment(seg.Geometry, point, minRel, maxRel, cfg.MaxSnapDistanceMeter); ok {
		return proj
	}
	return FindProjectionNearMeasure(seg.Geometry, point, target, window)
}

func (s *Snapper) enforceBranchLock(best *Candidate, point GPSPoint) *Candidate {
	if best == nil || !foldedSegmentBranchLockEnabled(s.config) {
		return best
	}
	lock := s.state.BranchLock
	if lock == nil || best.Segment.Order != lock.SegmentOrder {
		return best
	}

	window := branchLockSearchWindow(s.config)
	rel := best.Measure - best.Segment.FromMeasure
	if measureDelta(rel, lock.LockedRelMeasure) <= window {
		jump := DistanceMeter(best.SnappedPoint, lock.LockedPoint)
		jumpSlack := s.config.SnappedJumpSlackMeter
		if jumpSlack <= 0 {
			jumpSlack = DefaultRouteSnappedJumpSlackMeter
		}
		if jump <= jumpSlack*2 {
			return best
		}
	}

	proj := projectOntoSegment(best.Segment, point, s.state.LastPoint, s.state, s.config)
	candidate := s.candidateFromProjection(best.Segment, point, proj)
	return &candidate
}

func (s *Snapper) updateBranchLock(best *Candidate, point GPSPoint) {
	if best == nil || !foldedSegmentBranchLockEnabled(s.config) {
		return
	}

	seg := best.Segment
	if s.state.BranchLock != nil && s.state.BranchLock.SegmentOrder != seg.Order {
		s.state.BranchLock = nil
		s.state.BranchNormalTicks = 0
	}
	if s.state.LastBest != nil &&
		(s.state.LastBest.Segment.Order != seg.Order || s.state.LastBest.Segment.ID != seg.ID) {
		s.state.BranchLock = nil
		s.state.BranchNormalTicks = 0
	}

	viable := viableProjections(seg.Geometry, point.Point, s.config.MaxSnapDistanceMeter)
	rel := best.Measure - seg.FromMeasure
	lineIdx := projectionLineIndex(seg.Geometry, best.SnappedPoint, rel)

	if isFoldedSegment(viable, s.config) {
		if s.state.BranchLock == nil || s.state.BranchLock.SegmentOrder != seg.Order {
			s.state.BranchLock = &BranchLock{
				SegmentOrder:     seg.Order,
				SegmentID:        seg.ID,
				LockedRelMeasure: rel,
				LockedMeasure:    best.Measure,
				LockedLineIndex:  lineIdx,
				LockedPoint:      best.SnappedPoint,
				ViableCount:      len(viable),
			}
		} else {
			lock := s.state.BranchLock
			if best.Measure >= lock.LockedMeasure-backwardMeasureEpsilonM {
				lock.LockedMeasure = best.Measure
				lock.LockedRelMeasure = rel
				lock.LockedLineIndex = lineIdx
				lock.LockedPoint = best.SnappedPoint
				lock.ViableCount = len(viable)
			}
		}
		s.state.BranchNormalTicks = 0
	} else if s.state.BranchLock != nil && s.state.BranchLock.SegmentOrder == seg.Order {
		s.state.BranchNormalTicks++
		if s.state.BranchNormalTicks >= branchUnlockNormalTicks(s.config) {
			s.state.BranchLock = nil
			s.state.BranchNormalTicks = 0
		}
	} else {
		s.state.BranchLock = nil
		s.state.BranchNormalTicks = 0
	}

	s.recordBranchTick(seg.Order, lineIdx, rel)
	s.reinforceLockOnOscillation(seg, point)
}

func (s *Snapper) recordBranchTick(segmentOrder, lineIndex int, relMeasure float64) {
	tick := branchTick{
		SegmentOrder: segmentOrder,
		LineIndex:    lineIndex,
		Measure:      relMeasure,
	}
	s.state.RecentBranchTicks = append(s.state.RecentBranchTicks, tick)
	if len(s.state.RecentBranchTicks) > 5 {
		s.state.RecentBranchTicks = s.state.RecentBranchTicks[len(s.state.RecentBranchTicks)-5:]
	}
}

func (s *Snapper) reinforceLockOnOscillation(seg Segment, point GPSPoint) {
	ticks := s.state.RecentBranchTicks
	if len(ticks) < 3 {
		return
	}
	a := ticks[len(ticks)-3]
	b := ticks[len(ticks)-2]
	c := ticks[len(ticks)-1]
	if a.SegmentOrder != seg.Order || b.SegmentOrder != seg.Order || c.SegmentOrder != seg.Order {
		return
	}
	if a.LineIndex == b.LineIndex || b.LineIndex == c.LineIndex {
		return
	}
	if a.LineIndex != c.LineIndex {
		return
	}
	if measureDelta(b.Measure, a.Measure) < branchOscillationMeasureDeltaM {
		return
	}

	lock := s.state.BranchLock
	if lock == nil || lock.SegmentOrder != seg.Order {
		lock = &BranchLock{SegmentOrder: seg.Order, SegmentID: seg.ID}
		s.state.BranchLock = lock
	}
	lock.LockedRelMeasure = a.Measure
	lock.LockedMeasure = seg.FromMeasure + a.Measure
	lock.LockedLineIndex = a.LineIndex
	proj := projectOnLockedBranch(seg, point.Point, lock, s.config)
	lock.LockedPoint = proj.Point
	lock.LockedRelMeasure = proj.Measure
	lock.LockedMeasure = seg.FromMeasure + proj.Measure
}

func (s *Snapper) annotateBranchLock(result *SnapResult) {
	if result == nil || s.state.BranchLock == nil {
		return
	}
	if result.SegmentOrder == s.state.BranchLock.SegmentOrder {
		result.BranchLocked = true
		if result.HeldReason == "" {
			result.HeldReason = "branch_lock"
		}
	}
}
