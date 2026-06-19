# Changelog

## v0.2.1 — 2026-06-19

- fix: wrong snap on parallel lane at terminal overlap (`ProjectPointOnLineContinued`)
- fix: measure-continuity projection when folded geometry has multiple viable candidates
- feat: `MeasureAdvanceSlackMeter`, `SnappedJumpSlackMeter`, `SegmentSwitchHysteresisLog` in `RouteSnapConfig`
- fix: segment-switch hysteresis and overlap/lateral clamp for low-confidence snaps
- test: `TestParallelApproachDoesNotJumpToOffsetLane`

## v0.2.0 — 2026-06-19

- feat: `RouteSnapConfig` with optional params and functional options (`WithMeasureRegressionTolerance`, etc.)
- feat: backward segment rejection (`PreventBackwardTransition`) and measure regression guard
- feat: post-Viterbi backward clamp at terminal overlaps (`ClampBackwardMinConfidence`, `ClampDwellSpeedKmh`)
- feat: `Snapper.RouteMeasure`, `SnapResultFromSegment`
- test: terminal overlap backward guard and RouteSnapConfig defaults
- docs: README RouteSnapConfig section; deploy and viterbi agent skills

## v0.1.0 — 2026-06-17

- Initial release: GPS snap to linestring, gate-to-gate segments, Viterbi smoothing
- Loop route support, bearing/direction validation, parallel path discrimination
