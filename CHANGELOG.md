# Changelog

## v1.3.1 — 2026-07-10

- feat: `SnapResult` prev/curr/next stop context (`PrevStopID`, `PrevStopName`, `CurrStopID`, `CurrStopName`, `NextStopID`, `NextStopName`) for downstream map/ETA consumers

## v1.3.0 — 2026-07-10

- feat: teleport detection — freeze snap on implausible GPS movement within a short time window (`TeleportDetection`, default on in `RouteSnapConfig`)
- feat: GPS jump detection — classify expected-vs-actual movement ratio and hold on reject (`GpsJumpDetection`, `GpsJumpRatio`, `GpsJumpLevel`, `JumpCount`)
- feat: consecutive off-route detection — flag off-route after sustained lateral distance (`OffRouteDetection`, `OffRouteCount`)
- feat: reverse detection — tolerate small backward measure regression, hold until turnaround validated (`ReverseDetection`, `ReverseCount`, `TurnaroundValidated`)
- feat: segment sequence validation — reject skipped segment order jumps with recovery path (`SegmentSequenceValidation`, `SegmentJumpCount`, `SkippedSegmentCount`)
- feat: `RouteSnapConfig` options and defaults for all new detection modules (enabled by default for live routes)
- test: `teleport_detection_test`, `gps_jump_detection_test`, `off_route_detection_test`, `reverse_detection_test`, `segment_sequence_validation_test`

## v1.2.0 — 2026-07-06

- feat: `LiveBusSnapConfig(stops)` — production-tuned preset for live MQTT bus tracking (`Recommended*` constants, `IsLoopRoute`)
- feat: `OffRoutePolicy`, `DefaultOffRoutePolicy`, `MapOffRoute`, `EtaSnapReliableForPublish`, `SnapDegraded` — shared map/ETA off-route decisions
- feat: segment switch gated by halte radius (`RequireStopRadiusForSegmentSwitch`, `SegmentSwitchStopRadiusMeter`, default 20 m)
- feat: segment depart latch — advance to next segment after leaving junction stop off-route
- feat: immediate Viterbi reset when raw-to-snap distance exceeds `SnapDistanceResetMaxMeter` (default 100 m)
- docs: README recommended live-bus configuration section and off-route policy table
- test: `live_bus_config_test`, `off_route_policy_test`, segment-switch radius and snap-distance max reset

## v1.1.1 — 2026-06-30

- docs: README status reflects v1.1.0 release

## v1.1.0 — 2026-06-30

- feat: GPS drift resnap — re-project onto active segment when raw GPS drifts far from frozen snap but stays near the route (`resnapOnActiveSegmentWhenDrifted`)
- feat: Viterbi fallback `pickNearestForwardCandidate` when backward segment transition is rejected
- fix: stop forward creep when raw GPS drifts beyond max snap and is no longer near the route polyline (prevents measure racing ahead while lateral error grows)
- fix: reject backward segment transitions via `enforceForwardSegmentOrder`
- fix: default `MaxForwardSnapMeter` is 0 (disabled) — no per-tick forward snap cap unless explicitly configured
- test: `snap_drift_test` — drift resnap, backward segment guard, off-route creep regression

## v1.0.0 — 2026-06-22

- **Stable release** — public API (`Snapper`, `RouteSnapConfig`, `SnapResult`, segment/Viterbi helpers) is now semver-stable
- Production-ready live-bus snapping: Viterbi smoothing, backward guard, hold segment, wild GPS stabilize, forward cap, branch lock, segment switch guard, snap continuity, distance grow-reset
- MIT LICENSE (v0.2.9)

## v0.2.9 — 2026-06-22

- chore: add MIT LICENSE for pkg.go.dev and downstream redistribution

## v0.2.8 — 2026-06-22

- feat: snap continuity from previous position — limit geodesic/route jumps vs last snap using GPS movement + forward cap (`SnapContinuityFromPrevious`, default on)
- feat: ambiguous geometry via measure spread (>45 m between viable projections), not only ≥3 candidates
- feat: on ambiguous segments, pick projection closest to previous snap (not raw GPS alone)
- fix: forward creep when GPS moves but snap stays frozen (off-route drift / folded geometry) — bus advances along route instead of locking at last snap
- fix: creep respects route bearing and stays on current segment until stop passage
- feat: snap distance grow-reset — clears Viterbi state after 2 consecutive ticks of growing raw-to-snap distance (`SnapDistanceResetOnGrow`, default on)
- test: `TestSnapContinuityLimitsJumpFromPreviousSnap`, `TestSnapContinuityCreepsForwardWhenGPSMovesButSnapStuck`, `TestSnapDistanceResetWhenDistanceKeepsGrowing`

## v0.2.7 — 2026-06-22

- feat: folded-segment branch lock — when a segment has ≥3 viable projections, pick nearest GPS branch and pin until geometry normalizes (`FoldedSegmentBranchLock`, default on)
- feat: oscillation reinforcement — detect branch A→B→A flip-flop and re-pin to stable branch
- feat: `BranchLocked` on `SnapResult`; branch lock resets on segment switch
- test: `TestFoldedSegmentBranchLockPicksNearestAndStays`

## v0.2.6 — 2026-06-22

- feat: segment_id only changes after passing the current segment's destination stop (`RequireNextStopBeforeSegmentSwitch`, default on in `RouteSnapConfig`)
- feat: `NextStopPassToleranceMeter` (default 8 m) for route-measure slack at stop passage
- test: `TestSegmentSwitchBlockedBeforeNextStop`, `TestSegmentSwitchAllowedAfterNextStop`

## v0.2.5 — 2026-06-22

- feat: hold previous segment when no snap candidates match (`HoldLastSegmentOnMiss`, default on in `RouteSnapConfig`)
- feat: `HeldSegment` / `HeldReason` on `SnapResult` for downstream ETA consumers
- fix: hold no longer rejects GPS beyond 60 m; keeps segment with actual lateral distance (default hold projection 120 m)
- feat: `LastGood` Viterbi state survives brief GPS glitches
- feat: wild GPS jump stabilization — freeze on backward risk, cap forward advance (`WildGPSStabilize`)
- feat: max forward snap advance 50 m per tick along route line (`MaxForwardSnapMeter`)
- feat: strict no-backward snap — freeze at last position when measure/segment would regress (`NoBackwardSnap`)
- fix: no-backward uses forward creep along route when GPS progresses but snap projection regresses (bus no longer stuck)
- test: `TestHoldLastSegmentOnNoCandidates`, expiry, disabled, and far-distance cases
- test: `TestWildGPSJumpFreezesBackwardSnap`, `TestWildGPSJumpCapsForwardAdvance`

## v0.2.4 — 2026-06-19

- fix: prefer projection on active segment when Viterbi pick is far from raw GPS (`preferNearbyOnActiveSegment`)
- fix: tighter segment-switch hysteresis distance (12 m ambiguous window)
- dashboard: default max snap 28 m, bearing validation 40°, stricter off-route at 22 m / 15 m + low confidence

## v0.2.3 — 2026-06-19

- fix: clamp allows forward measure progress when GPS moves along segment (no sticky snap)
- fix: limit same-segment lateral jump per GPS tick (`stabilizeSameSegmentCandidate`)
- feat: `ForwardProjectionOnSegment` for forward-only measure window on folded geometry
- test: `TestClampAllowsForwardProgressAlongSegment`

## v0.2.2 — 2026-06-19

- fix: speed-based direction weakening compared km/h to m/s (`shouldWeakenDirectionValidation`)
- fix: dwell / stationary GPS keeps measure-continuity projection on folded segment geometry
- fix: parallel-branch tie-break uses last snapped position, not raw GPS distance
- fix: `ClampDwellSpeedKmh` wired into lateral backward clamp on low-speed snaps
- test: `TestStationaryGPSDoesNotJumpOnFoldedSegment`

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
