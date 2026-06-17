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

    snapper, err := snaptoline.NewSnapper(line, stops, snaptoline.DefaultConfig())
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

Use `snaptoline.DefaultConfig()` as a starting point and override fields as needed.

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

Example for a looping route:

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
| `Config()` | Return current configuration |

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

## Status

Experimental. API may change before `v1.0.0`.
