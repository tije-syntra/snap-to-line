# snap-to-line

A Go library for snapping live bus GPS positions onto route linestrings. Built for real-world transit tracking where GPS is noisy, routes loop, and outbound/inbound paths run in parallel.

**Repository:** [github.com/tije-syntra/snap-to-line](https://github.com/tije-syntra/snap-to-line)

## Features

- **Gate-to-gate segments** — builds route segments from ordered stops
- **Viterbi smoothing** — keeps snapping stable across noisy GPS updates
- **Bearing validation** — avoids snapping to the wrong parallel path
- **Trip direction** — prefers outbound or inbound segments when configured
- **Loop routes** — handles routes where the first and last stop share the same ID and coordinates
- **Measure-based geometry** — slices linestrings by linear distance, not raw coordinates

## Installation

```bash
go get github.com/tije-syntra/snap-to-line
```

Requires **Go 1.22+**.

## Quick start

```go
package main

import (
    "fmt"
    "log"

    "github.com/paulmach/orb"
    snaptoline "github.com/tije-syntra/snap-to-line"
)

func main() {
    line := orb.LineString{
        {106.0, -6.0},
        {106.05, -6.0},
        {106.1, -6.0},
    }

    stops := []snaptoline.Stop{
        {ID: "A", Order: 1, Point: line[0]},
        {ID: "B", Order: 2, Point: line[1]},
        {ID: "C", Order: 3, Point: line[2]},
    }

    snapper, err := snaptoline.NewSnapper(line, stops, snaptoline.LiveBusSnapConfig(stops))
    if err != nil {
        log.Fatal(err)
    }

    result, err := snapper.Snap(snaptoline.GPSPoint{
        Point:   orb.Point{106.04, -6.0001},
        Bearing: 90,
        Speed:   8,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("segment=%s distance=%.1fm confidence=%.2f\n",
        result.SegmentID, result.DistanceMeter, result.Confidence)
}
```

Run the included example:

```bash
go run ./examples/basic
```

## How it works

```
Route line + ordered stops
        │
        ▼
Sequential stop projection (monotonic measure)
        │
        ▼
Gate-to-gate segments
        │
        ▼
GPS point → candidate segments
        │
        ▼
Score: distance × bearing × trip direction
        │
        ▼
Viterbi (transition + emission)
        │
        ▼
SnapResult
```

Each GPS update produces candidates on nearby segments. Candidates are scored by distance to the line, alignment between bus bearing and line bearing, and whether the segment matches the active trip direction. The Viterbi step combines these scores with transition probabilities so the bus does not jump to distant or backwards segments.

## Configuration

For **live bus tracking** (MQTT GPS, map markers, ETA), use `LiveBusSnapConfig(stops)` — see [Recommended live-bus configuration](#recommended-live-bus-configuration).

For prototyping or custom pipelines, start from `DefaultConfig()` and override fields as needed.

| Field | Default | Description |
|-------|---------|-------------|
| `MaxSnapDistanceMeter` | `60` | Maximum perpendicular distance from the route to accept a snap |
| `CandidateLimit` | `8` | Maximum number of segment candidates per GPS point |
| `Looping` | `false` | Enable loop-route handling |
| `UseBearingValidation` | `true` | Penalize candidates whose bearing disagrees with the bus |
| `MaxBearingDiffDegree` | `60` | Maximum allowed bearing difference before heavy penalty |
| `UseMovementBearing` | `true` | Derive bearing from the previous GPS point when not provided |
| `MinMovementMeter` | `8` | Minimum movement required to compute movement-based bearing |
| `UseTripDirection` | `false` | Enable outbound/inbound segment preference |
| `TripDirection` | `unknown` | Active trip direction (`outbound`, `inbound`, `loop`, `unknown`) |
| `AllowSameStartEndStop` | `true` | Treat identical first/last stops as separate occurrences on loops |
| `LoopClosureToleranceMeter` | `10` | Distance tolerance when comparing first and last stop |
| `UseSpeed` | `true` | Weaken bearing validation when speed is very low |
| `SegmentDirections` | `nil` | Optional per-segment direction override for parallel routes |
| `PreventBackwardTransition` | `false` | Reject Viterbi candidates on lower segment order (except loop wrap) |
| `MeasureRegressionToleranceMeter` | `0` | Reject candidates whose route measure drops more than this |
| `ClampBackwardMinConfidence` | `0` | Post-Viterbi clamp when backward slip has confidence below this (`0` = off) |
| `ClampDwellSpeedKmh` | `0` | Speed at or below this treated as dwell when clamping (`0` = use 8 km/h) |

For live bus tracking, use `RouteSnapConfig(stops, opts...)` which enables backward guards and auto-detects loop routes when first and last stop match. All route-specific settings are optional; omitted options use the defaults below.

### `RouteSnapConfig` defaults

| Parameter | Default | Description |
|-----------|---------|-------------|
| `PreventBackwardTransition` | `true` | Reject Viterbi candidates on lower segment order |
| `MeasureRegressionToleranceMeter` | `30` | Reject snap when route measure drops more than this (meters) |
| `ClampBackwardMinConfidence` | `0.55` | Post-Viterbi clamp when backward slip has confidence below this; `0` disables clamp |
| `ClampDwellSpeedKmh` | `8` | Speed ≤ this (km/h) treated as dwell when clamping at terminals |
| `MeasureAdvanceSlackMeter` | `15` | Cap unrealistic forward measure jumps vs GPS movement on folded geometry |
| `SnappedJumpSlackMeter` | `4` | Cap lateral snap jumps vs GPS movement on overlapping branches |
| `SegmentSwitchHysteresisLog` | `1.0` | Minimum log-score margin to change segment in ambiguous zones (`0` disables) |
| `Looping` | auto | `true` when first and last stop share the same ID/coords within tolerance; override with `WithLooping` |
| `LoopClosureToleranceMeter` | `10` | Tolerance for same start/end stop detection (from `DefaultConfig`) |

Constants: `DefaultRouteMeasureRegressionToleranceMeter`, `DefaultRouteClampBackwardMinConfidence`, `DefaultRouteClampDwellSpeedKmh`, `DefaultRouteMeasureAdvanceSlackMeter`, `DefaultRouteSnappedJumpSlackMeter`, `DefaultRouteSegmentSwitchHysteresisLog`.

## Recommended live-bus configuration

`LiveBusSnapConfig(stops)` bundles production-tuned values used by [snap-to-line-dashboard](https://github.com/tije-syntra/snap-to-line-dashboard) for real-time bus tracking. It is stricter than bare `RouteSnapConfig` defaults: tighter snap radius, stronger backward guards, and segment-switch gates at stops.

```go
cfg := snaptoline.LiveBusSnapConfig(stops)
snapper, err := snaptoline.NewSnapper(line, stops, cfg)
```

Pair with `DefaultOffRoutePolicy()` for map/UI off-route and ETA freeze decisions:

```go
policy := snaptoline.DefaultOffRoutePolicy()
result, _ := snapper.Snap(point)
degraded := snaptoline.SnapDegraded(result)
offRoute := snaptoline.MapOffRoute(result, degraded, policy)
etaOK := snaptoline.EtaSnapReliableForPublish(result, degraded, policy)
```

### Snap settings (`LiveBusSnapConfig`)

| Setting | Value | Notes |
|---------|-------|-------|
| `MaxSnapDistanceMeter` | `28` | Reject snaps farther than 28 m from the route |
| `MaxBearingDiffDegree` | `40` | Stricter bearing gate than `DefaultConfig` (60°) |
| `MinMovementMeter` | `3` | Lower movement threshold for movement-based bearing |
| `PreventBackwardTransition` | `true` | Viterbi cannot jump to lower segment order |
| `MeasureRegressionToleranceMeter` | `10` | Reject measure regressions > 10 m (vs 30 m default) |
| `ClampBackwardMinConfidence` | `0.78` | Post-Viterbi clamp on low-confidence backward slip |
| `ClampDwellSpeedKmh` | `10` | Treat ≤ 10 km/h as dwell when clamping at terminals |
| `MeasureAdvanceSlackMeter` | `8` | Cap unrealistic forward measure jumps on folded geometry |
| `SnappedJumpSlackMeter` | `2` | Cap lateral snap jumps vs GPS movement |
| `SegmentSwitchHysteresisLog` | `2.5` | Require stronger score margin to switch segment |
| `RequireNextStopBeforeSegmentSwitch` | `true` | Bus must pass next stop before switching segment |
| `RequireStopRadiusForSegmentSwitch` | `true` | Segment switch only inside stop radius |
| `SegmentSwitchStopRadiusMeter` | `20` | Stop-radius gate for segment switches |
| `FoldedSegmentBranchLock` | `true` | Stabilize overlapping outbound/inbound branches |
| `SnapContinuityFromPrevious` | `true` | Prefer continuity with previous snap on folded routes |
| `SnapDistanceResetMaxMeter` | `100` | Hard reset when lateral distance exceeds 100 m |
| `LoopClosureToleranceMeter` | `15` | Detect loop when first/last stop within 15 m |

**Non-loop routes** also enable grow-reset: after 2 consecutive ticks with snap distance growing by ≥ 8 m and above 35 m, the snapper resets Viterbi state.

**Loop routes** (`IsLoopRoute(stops) == true`) disable grow-reset and set `NextStopPassToleranceMeter` to `18` m for smoother closure at the repeated terminal stop.

All values are exported as `Recommended*` constants (e.g. `RecommendedMaxSnapDistanceMeter`) for partial overrides:

```go
cfg := snaptoline.LiveBusSnapConfig(stops)
cfg.MaxSnapDistanceMeter = 35 // override one field after the preset
```

### Off-route policy (`DefaultOffRoutePolicy`)

| Setting | Value | Effect |
|---------|-------|--------|
| `MaxSnapDistanceMeter` | `28` | Aligns with snap max distance |
| `OffRouteSoftDistFraction` | `0.54` | Soft off-route at ~15 m (28 × 0.54) when confidence is low |
| `OffRouteMinConfidence` | `0.2` | Below this → always off-route on map |
| `OffRouteSoftConfidence` | `0.5` | Below this + beyond soft distance → off-route |

`MapOffRoute` respects held-segment masking (ETA hold keeps the map marker on-route). `EtaSnapReliableForPublish` returns `false` when snap is degraded or off-route by policy.

### Usage examples

Recommended preset (live bus):

```go
cfg := snaptoline.LiveBusSnapConfig(stops)
snapper, err := snaptoline.NewSnapper(line, stops, cfg)
```

`RouteSnapConfig` with library defaults (looser than live-bus preset):

```go
cfg := snaptoline.RouteSnapConfig(stops)
snapper, err := snaptoline.NewSnapper(line, stops, cfg)
```

Functional options (manual tuning):

```go
cfg := snaptoline.RouteSnapConfig(stops,
    snaptoline.WithMeasureRegressionTolerance(40),
    snaptoline.WithClampDwellSpeedKmh(5),
    snaptoline.WithLooping(true),
)
```

Params struct (e.g. from env/API); nil fields keep defaults:

```go
tolerance := 40.0
dwellKmh := 6.0
cfg := snaptoline.RouteSnapConfig(stops, snaptoline.RouteSnapParamsOption(snaptoline.RouteSnapParams{
    MeasureRegressionToleranceMeter: &tolerance,
    ClampDwellSpeedKmh:              &dwellKmh,
}))
```

Disable post-Viterbi backward clamp:

```go
cfg := snaptoline.RouteSnapConfig(stops, snaptoline.DisableBackwardClamp())
```

Available option helpers: `WithPreventBackwardTransition`, `WithMeasureRegressionTolerance`, `WithClampBackwardMinConfidence`, `WithClampDwellSpeedKmh`, `WithMeasureAdvanceSlack`, `WithSnappedJumpSlack`, `WithSegmentSwitchHysteresisLog`, `WithLooping`, `WithLoopClosureTolerance`, `WithRouteSnapParams`, `RouteSnapParamsOption`, `DisableBackwardClamp`.

Example for a looping route (manual config without `RouteSnapConfig`):

```go
cfg := snaptoline.Config{
    Looping:                   true,
    AllowSameStartEndStop:     true,
    LoopClosureToleranceMeter: 10,
    MaxSnapDistanceMeter:      60,
    UseBearingValidation:      true,
    MaxBearingDiffDegree:      60,
}
```

## Loop routes

On a loop such as `A → B → C → A`, the first and last stop may share the same ID and coordinates. The library treats them as two different **occurrences** along the line:

```
A (occurrence 1, measure 0)
B (measure 1200)
C (measure 2500)
A (occurrence 2, measure 3600, loop closure)
```

Segments are sliced from measure to measure. The closing segment `C → A` runs from the last intermediate stop to the **end of the linestring**, not back to the first occurrence of `A`.

## Parallel outbound / inbound paths

When outbound and inbound paths are close together, the nearest geometric candidate is not always correct. Configure trip direction and optionally assign per-segment directions:

```go
cfg := snaptoline.Config{
    UseTripDirection:     true,
    TripDirection:        snaptoline.DirectionOutbound,
    UseBearingValidation: true,
    MaxBearingDiffDegree: 60,
    SegmentDirections: []snaptoline.DirectionType{
        snaptoline.DirectionOutbound,
        snaptoline.DirectionOutbound,
        snaptoline.DirectionInbound,
    },
}

snapper, err := snaptoline.NewSnapper(line, stops, cfg)
```

You can change direction at runtime:

```go
snapper.SetTripDirection(snaptoline.DirectionInbound)
```

## API

### Snapper

| Method | Description |
|--------|-------------|
| `NewSnapper(line, stops, cfg)` | Build segments and initialize Viterbi state |
| `Snap(point)` | Snap a GPS point and return `*SnapResult` |
| `Reset()` | Clear Viterbi history |
| `SetTripDirection(direction)` | Update active trip direction |
| `Segments()` | Return a copy of built segments |
| `RouteMeasure(order, progress)` | Cumulative distance along route for segment order + progress |
| `Config()` | Return current configuration |

### Route snap config

| Function | Description |
|----------|-------------|
| `LiveBusSnapConfig(stops)` | **Recommended** production preset for live bus tracking |
| `IsLoopRoute(stops)` | Detect loop routes (first/last stop within 15 m) |
| `RouteSnapConfig(stops, opts...)` | Route-aware defaults with optional overrides |
| `DefaultOffRoutePolicy()` | Off-route / ETA thresholds aligned with live-bus snap |
| `MapOffRoute`, `EtaSnapReliableForPublish`, `SnapDegraded` | Post-snap policy helpers |
| `RouteSnapParams` | Struct of optional pointer fields for overrides |
| `WithMeasureRegressionTolerance`, etc. | Functional option helpers |

See [Recommended live-bus configuration](#recommended-live-bus-configuration) and [RouteSnapConfig defaults](#routesnapconfig-defaults).

### SnapResult

Key fields returned from `Snap`:

| Field | Description |
|-------|-------------|
| `SnappedPoint` | Projected position on the route |
| `SegmentID` / `SegmentOrder` | Matched segment |
| `DistanceMeter` | Perpendicular distance from GPS to route |
| `Progress` | Position along the segment, `0.0`–`1.0` |
| `Confidence` | Combined score confidence, `0.0`–`1.0` |
| `IsOffRoute` | `true` when no valid candidate is within range |
| `BusBearing` / `LineBearing` / `DirectionDiff` | Bearing diagnostics |

### Lower-level helpers

These are exported for advanced use and testing:

- `ProjectStopsSequential` — project stops onto a line using monotonic measure
- `BuildSegments` / `BuildSegmentsFromProjectedStops` — build gate-to-gate segments
- `SliceLineByMeasure` — extract a sub-line between two measures
- `BearingDiff`, `DirectionScore`, `TripDirectionScore`, `TransitionScore`

## Testing

```bash
go test ./...
```

Tests cover:

- Basic snapping and off-route detection
- Viterbi stability under GPS noise
- Loop routes with identical start/end stops
- Measure-based segment slicing
- Parallel outbound/inbound path discrimination
- Bearing wrap-around (e.g. 350° vs 10°)

## Project layout

```
.
├── snapper.go          # Public API
├── live_bus_config.go  # LiveBusSnapConfig production preset
├── off_route_policy.go # Off-route and ETA reliability policy
├── segment.go          # Segment builder
├── loop.go             # Sequential stop projection
├── viterbi.go          # Candidate scoring and Viterbi
├── direction.go        # Bearing and direction scoring
├── geometry.go         # Geometry helpers
├── internal/mathgeo/   # Distance, bearing, projection, measure
├── examples/basic/     # Runnable example
└── tests/              # Integration-style tests
```

## Dependencies

- [github.com/paulmach/orb](https://github.com/paulmach/orb) — geometry types (`orb.Point`, `orb.LineString`)

## License

MIT License — see [LICENSE](LICENSE). Redistributable with minimal restrictions on use, modification, and redistribution.

## Status

**v1.2.0** — stable. Breaking API changes will require a new major version (`/v2` module path per Go convention).
