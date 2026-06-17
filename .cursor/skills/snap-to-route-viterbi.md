# Skill: Go Bus Snap to Linestring with Viterbi, Direction Validation, and Loop Stop Handling

## Tujuan

Membuat library Go untuk melakukan **snap posisi bus ke linestring rute**, menggunakan:

* Segment gate-to-gate
* Algoritma Viterbi
* Validasi arah bus terhadap arah line
* Validasi trip direction berangkat/pulang
* Handling rute looping
* Handling stop awal sama dengan stop akhir
* Handling line sejajar atau beririsan

Library ini dibuat untuk kasus **bus live tracking**, terutama ketika rute memiliki jalur berangkat dan pulang yang sejajar, GPS tidak akurat, dan linestring looping.

---

## Masalah yang Diselesaikan

### 1. GPS Noise

GPS bus bisa melenceng beberapa meter dari jalur asli.

### 2. Jalur Sejajar

Contoh:

```txt
Jalur berangkat: arah timur
Jalur pulang:    arah barat
```

Jika GPS lebih dekat ke jalur pulang, snap biasa bisa salah.

### 3. Linestring Looping

Pada rute looping, stop awal bisa sama dengan stop akhir.

Contoh:

```txt
Stop A -> Stop B -> Stop C -> Stop A
```

Masalahnya:

```txt
Stop awal ID = Stop akhir ID
Koordinat stop awal = koordinat stop akhir
```

Jika segment builder hanya mencari stop berdasarkan ID atau koordinat pertama yang cocok, maka slice bisa salah.

Contoh salah:

```txt
Segment C -> A harusnya mengambil bagian akhir linestring.
Tapi karena A juga ada di awal, segment C -> A malah slice ke titik A pertama.
```

Akibatnya:

```txt
Segment menjadi kosong
Segment terbalik
Segment terlalu panjang
Segment salah melewati seluruh route
```

---

## Nama Library

```txt
snap-to-line
```

Go module:

```bash
go mod init github.com/tije-syntra/snap-to-line
```

---

## Dependency

```bash
go get github.com/paulmach/orb
```

Opsional:

```bash
go get github.com/stretchr/testify
```

---

## Struktur Folder

```txt
snap-to-line/
├── go.mod
├── README.md
├── SKILL.md
├── CHANGELOG.md
├── LICENSE
├── types.go
├── snapper.go
├── segment.go
├── geometry.go
├── direction.go
├── viterbi.go
├── loop.go
├── errors.go
├── internal/
│   └── mathgeo/
│       ├── distance.go
│       ├── bearing.go
│       ├── projection.go
│       └── measure.go
├── examples/
│   └── basic/
│       └── main.go
└── tests/
    ├── snapper_test.go
    ├── segment_test.go
    ├── loop_segment_test.go
    ├── viterbi_test.go
    ├── direction_test.go
    └── looping_test.go
```

---

## Prinsip Penting

Untuk membuat segment pada linestring looping, jangan hanya mengandalkan:

```txt
stop_id
koordinat stop
nearest point pertama
```

Gunakan:

```txt
stop order
projected measure
occurrence index
monotonic progression
```

Artinya, setiap stop harus diproyeksikan ke posisi tertentu di sepanjang linestring berdasarkan urutan perjalanan, bukan hanya berdasarkan jarak terdekat secara global.

---

## Tipe Data

```go
package snaptoline

import "github.com/paulmach/orb"

type DirectionType string

const (
	DirectionOutbound DirectionType = "outbound"
	DirectionInbound  DirectionType = "inbound"
	DirectionLoop     DirectionType = "loop"
	DirectionUnknown  DirectionType = "unknown"
)

type Stop struct {
	ID    string
	Name  string
	Point orb.Point
	Order int
}

type ProjectedStop struct {
	Stop          Stop
	Measure       float64
	LineIndex     int
	Occurrence    int
	IsLoopClosure bool
}

type Segment struct {
	ID            string
	FromStop      Stop
	ToStop        Stop
	FromMeasure   float64
	ToMeasure     float64
	Geometry      orb.LineString
	Order         int
	Direction     DirectionType
	Bearing       float64
	IsLooping     bool
	IsLoopClosing bool
}

type GPSPoint struct {
	Point     orb.Point
	Speed     float64
	Bearing   float64
	Timestamp int64
}

type SnapResult struct {
	OriginalPoint  orb.Point
	SnappedPoint   orb.Point
	SegmentID      string
	SegmentOrder   int
	Direction      DirectionType
	NearestStopID  string
	DistanceMeter  float64
	Progress       float64
	BusBearing     float64
	LineBearing    float64
	DirectionDiff  float64
	Confidence     float64
	IsOffRoute      bool
	RejectedReason string
}
```

---

## Config

```go
type Config struct {
	MaxSnapDistanceMeter float64
	CandidateLimit      int

	Looping bool

	UseBearingValidation bool
	MaxBearingDiffDegree float64

	UseMovementBearing bool
	MinMovementMeter    float64

	UseTripDirection bool
	TripDirection    DirectionType

	AllowSameStartEndStop bool
	LoopClosureToleranceMeter float64

	UseSpeed bool
}
```

Contoh:

```go
cfg := snaptoline.Config{
	MaxSnapDistanceMeter: 60,
	CandidateLimit:      8,

	Looping: true,

	UseBearingValidation: true,
	MaxBearingDiffDegree:  60,

	UseMovementBearing: true,
	MinMovementMeter:    8,

	UseTripDirection: true,
	TripDirection:    snaptoline.DirectionOutbound,

	AllowSameStartEndStop: true,
	LoopClosureToleranceMeter: 10,

	UseSpeed: true,
}
```

---

## Public API

```go
type Snapper struct {
	segments []Segment
	stops    []Stop
	config   Config
	state    *ViterbiState
}

func NewSnapper(
	line orb.LineString,
	stops []Stop,
	cfg Config,
) (*Snapper, error)

func (s *Snapper) Snap(point GPSPoint) (*SnapResult, error)

func (s *Snapper) Reset()

func (s *Snapper) SetTripDirection(direction DirectionType)
```

---

## Loop Stop Problem

Input stops:

```txt
A, B, C, A
```

Dengan detail:

```txt
Stop A order 1
Stop B order 2
Stop C order 3
Stop A order 4
```

Walaupun `StopID` dan koordinat A sama, library harus menganggapnya sebagai dua occurrence berbeda:

```txt
A occurrence 1 = awal route
A occurrence 2 = akhir route
```

Jangan merge stop hanya karena ID sama.

---

## Validasi Stop Looping

Jika:

```txt
Looping = true
stop pertama ID == stop terakhir ID
koordinat stop pertama == stop terakhir
```

Maka:

```txt
stop terakhir harus ditandai sebagai IsLoopClosure = true
occurrence stop terakhir harus lebih besar dari occurrence stop pertama
measure stop terakhir harus berada di akhir linestring
```

Contoh:

```go
func IsSameStop(a, b Stop, toleranceMeter float64) bool {
	if a.ID != b.ID {
		return false
	}

	return DistanceMeter(a.Point, b.Point) <= toleranceMeter
}
```

---

## Projected Stop Logic

### Salah

```go
for _, stop := range stops {
	projected := FindNearestPointOnLine(line, stop.Point)
}
```

Masalah:

```txt
Stop A awal dan A akhir akan sama-sama diproyeksikan ke titik A pertama.
```

### Benar

Gunakan sequential projection.

```go
func ProjectStopsSequential(
	line orb.LineString,
	stops []Stop,
	cfg Config,
) ([]ProjectedStop, error)
```

Aturan:

```txt
1. Stop diproses berdasarkan Order.
2. Projection search dimulai dari measure terakhir.
3. Stop berikutnya tidak boleh mundur dari stop sebelumnya.
4. Untuk stop terakhir yang sama dengan stop pertama pada route looping, cari occurrence di akhir line.
5. Jika line tertutup, stop terakhir boleh map ke total length line.
```

---

## Measure-Based Projection

Setiap titik pada linestring punya posisi linear:

```txt
measure = jarak dari awal line ke titik proyeksi
```

Contoh:

```txt
A awal = measure 0
B      = measure 1200
C      = measure 2500
A akhir = measure 3600
```

Walaupun A awal dan A akhir koordinatnya sama:

```txt
A awal != A akhir
```

Karena:

```txt
measure berbeda
```

---

## Logic Projection untuk Stop Awal = Stop Akhir

```go
func ProjectStopsSequential(line orb.LineString, stops []Stop, cfg Config) ([]ProjectedStop, error) {
	if len(stops) < 2 {
		return nil, ErrInsufficientStops
	}

	totalLength := LineLengthMeter(line)
	result := make([]ProjectedStop, 0, len(stops))

	firstStop := stops[0]
	lastStop := stops[len(stops)-1]

	isClosedLoopStop := cfg.Looping &&
		cfg.AllowSameStartEndStop &&
		IsSameStop(firstStop, lastStop, cfg.LoopClosureToleranceMeter)

	lastMeasure := 0.0

	for i, stop := range stops {
		var projected ProjectedStop

		if i == 0 {
			projected = ProjectStopNearMeasure(line, stop, 0, cfg)
			projected.Measure = 0
			projected.Occurrence = 1
		} else if isClosedLoopStop && i == len(stops)-1 {
			projected = ProjectStopNearMeasure(line, stop, totalLength, cfg)
			projected.Measure = totalLength
			projected.Occurrence = 2
			projected.IsLoopClosure = true
		} else {
			projected = ProjectStopForwardOnly(line, stop, lastMeasure, cfg)
			projected.Occurrence = CountOccurrence(result, stop.ID) + 1
		}

		if projected.Measure < lastMeasure {
			return nil, ErrStopMeasureNotMonotonic
		}

		result = append(result, projected)
		lastMeasure = projected.Measure
	}

	return result, nil
}
```

---

## ProjectStopForwardOnly

```go
func ProjectStopForwardOnly(
	line orb.LineString,
	stop Stop,
	minMeasure float64,
	cfg Config,
) ProjectedStop {
	candidates := FindProjectionCandidates(line, stop.Point)

	var best *ProjectionCandidate

	for _, c := range candidates {
		if c.Measure < minMeasure {
			continue
		}

		if best == nil || c.DistanceMeter < best.DistanceMeter {
			best = &c
		}
	}

	if best == nil {
		best = FindNearestProjectionAfterMeasure(line, stop.Point, minMeasure)
	}

	return ProjectedStop{
		Stop:      stop,
		Measure:   best.Measure,
		LineIndex: best.LineIndex,
	}
}
```

---

## Segment Builder

Setelah stops diproyeksikan, segment dibuat dari measure ke measure.

```go
func BuildSegmentsFromProjectedStops(
	line orb.LineString,
	projectedStops []ProjectedStop,
	cfg Config,
) ([]Segment, error)
```

Aturan:

```txt
1. Segment dibuat dari projectedStops[i] ke projectedStops[i+1].
2. Slice line berdasarkan FromMeasure dan ToMeasure.
3. Jika FromMeasure == ToMeasure, segment dianggap invalid.
4. Untuk loop closing segment, ToMeasure boleh sama dengan total line length.
5. Segment tidak boleh slice ke occurrence stop yang salah.
```

---

## Slice Berdasarkan Measure

Jangan slice berdasarkan koordinat stop langsung.

Salah:

```go
SliceLineByPoint(fromStop.Point, toStop.Point)
```

Benar:

```go
SliceLineByMeasure(line, fromMeasure, toMeasure)
```

Contoh:

```go
func SliceLineByMeasure(
	line orb.LineString,
	fromMeasure float64,
	toMeasure float64,
) (orb.LineString, error) {
	if toMeasure <= fromMeasure {
		return nil, ErrInvalidMeasureRange
	}

	// Ambil bagian line dari fromMeasure sampai toMeasure.
	// Interpolasi titik awal dan akhir jika measure jatuh di tengah segment.
	return sliced, nil
}
```

---

## Logic Loop Closing Segment

Untuk:

```txt
A -> B -> C -> A
```

Segment:

```txt
SEG-A-B: from measure 0    to 1200
SEG-B-C: from measure 1200 to 2500
SEG-C-A: from measure 2500 to 3600
```

Bukan:

```txt
SEG-C-A: from measure 2500 to 0
```

Dan bukan:

```txt
SEG-C-A: from measure 2500 to titik A pertama
```

---

## Validasi Arah Bus terhadap Arah Line

Digunakan untuk menghindari salah snap ke jalur sejajar.

### Hitung Bearing Bus

Gunakan salah satu:

```txt
1. Bearing dari GPS
2. Bearing dari previous GPS point ke current GPS point
3. Last known bearing
```

Jika bus berhenti atau speed rendah, lemahkan validasi arah.

---

### Hitung Bearing Segment

Bearing segment dihitung dari arah line di sekitar snapped point.

```go
lineBearing := BearingAtMeasure(segment.Geometry, candidate.Measure)
```

Jangan hanya pakai bearing awal-ke-akhir jika segment panjang dan berbelok.

---

### Bearing Diff

```go
func BearingDiff(a, b float64) float64 {
	diff := math.Abs(a - b)
	if diff > 180 {
		diff = 360 - diff
	}
	return diff
}
```

---

## Direction Score

```go
func DirectionScore(diff float64, maxDiff float64) float64 {
	if diff > maxDiff {
		return 0.05
	}

	return 1 - (diff / maxDiff)
}
```

---

## Trip Direction

Segment memiliki direction:

```go
DirectionOutbound
DirectionInbound
DirectionLoop
DirectionUnknown
```

Jika bus sedang berangkat:

```go
snapper.SetTripDirection(DirectionOutbound)
```

Maka segment inbound diberi penalty besar.

```go
func TripDirectionScore(candidate DirectionType, active DirectionType) float64 {
	if active == DirectionUnknown {
		return 1
	}

	if candidate == active {
		return 1
	}

	if candidate == DirectionLoop {
		return 0.7
	}

	return 0.05
}
```

---

## Candidate Scoring

```txt
final_score =
  emission_score *
  direction_score *
  trip_direction_score
```

Dalam Viterbi:

```txt
score =
  previous_score +
  log(transition_score) +
  log(emission_score) +
  log(direction_score) +
  log(trip_direction_score)
```

---

## Viterbi State

```go
type ViterbiState struct {
	LastCandidates  []Candidate
	LastBest        *Candidate
	LastPoint       *GPSPoint
	LastTimestamp   int64
	ActiveDirection DirectionType
}
```

---

## Transition Score

Aturan:

```txt
Sama segment              -> tinggi
Next segment              -> tinggi
Loncat 1-2 segment         -> sedang
Loncat terlalu jauh        -> rendah
Mundur segment             -> rendah
Last -> first saat looping -> valid
```

Untuk looping:

```txt
Segment terakhir boleh lanjut ke segment pertama
```

Namun untuk segment builder:

```txt
Loop closure tetap harus menggunakan measure akhir line, bukan measure awal.
```

---

## Handling GPS Noise

Jika:

```txt
Speed < 3 km/h
perpindahan < MinMovementMeter
bearing tidak tersedia
```

Maka:

```txt
direction validation dilemahkan
transition score lebih dominan
last known direction digunakan
```

---

## Basic Flow

```txt
Input line, stops, config
    ↓
Sort stops by order
    ↓
Detect loop closure stop
    ↓
Project stops sequentially by measure
    ↓
Build segment by measure
    ↓
Input GPS point
    ↓
Find candidate segment
    ↓
Calculate snapped point
    ↓
Calculate distance score
    ↓
Calculate line bearing
    ↓
Calculate bus bearing
    ↓
Calculate direction score
    ↓
Calculate trip direction score
    ↓
Run Viterbi
    ↓
Return SnapResult
```

---

## Testing Wajib

Minimal test:

1. Snap normal ke linestring.
2. Snap ke segment terdekat.
3. GPS jauh menghasilkan `IsOffRoute = true`.
4. Bus tidak loncat ke segment jauh.
5. Looping route bisa dari segment terakhir ke segment pertama.
6. Jalur sejajar berangkat dan pulang.
7. Bus berangkat tidak boleh tersnap ke jalur pulang.
8. Bus pulang tidak boleh tersnap ke jalur berangkat.
9. Bearing 90° vs line 270° harus diberi penalty besar.
10. Bearing 350° vs line 10° harus dianggap beda 20°.
11. Stop awal dan akhir dengan ID sama harus dianggap occurrence berbeda.
12. Stop awal dan akhir dengan koordinat sama tidak boleh membuat segment kosong.
13. Segment terakhir pada loop harus slice dari stop sebelum akhir ke total line length.
14. Segment C -> A pada loop tidak boleh slice ke A pertama.
15. Projected stop measure harus monotonic naik.
16. Slice segment harus berdasarkan measure, bukan koordinat.
17. Saat bus berhenti, direction validation tidak boleh terlalu agresif.
18. Saat GPS noise tinggi, Viterbi tetap memilih segment stabil.
19. Reset state menghapus riwayat Viterbi.

---

## Contoh Test Loop Stop

```go
func TestLoopStartEndSameStopShouldCreateValidClosingSegment(t *testing.T) {
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: orb.Point{106.0, -6.0}},
		{ID: "B", Order: 2, Point: orb.Point{106.1, -6.0}},
		{ID: "C", Order: 3, Point: orb.Point{106.1, -6.1}},
		{ID: "A", Order: 4, Point: orb.Point{106.0, -6.0}},
	}

	cfg := snaptoline.Config{
		Looping: true,
		AllowSameStartEndStop: true,
		LoopClosureToleranceMeter: 10,
	}

	projected, err := snaptoline.ProjectStopsSequential(line, stops, cfg)
	require.NoError(t, err)

	require.Equal(t, "A", projected[0].Stop.ID)
	require.Equal(t, "A", projected[3].Stop.ID)

	require.Equal(t, 1, projected[0].Occurrence)
	require.Equal(t, 2, projected[3].Occurrence)

	require.True(t, projected[3].IsLoopClosure)
	require.Greater(t, projected[3].Measure, projected[2].Measure)
}
```

---

## Contoh Test Segment Closing

```go
func TestClosingSegmentShouldUseEndMeasureNotFirstStopMeasure(t *testing.T) {
	segments, err := snaptoline.BuildSegmentsFromProjectedStops(line, projectedStops, cfg)
	require.NoError(t, err)

	last := segments[len(segments)-1]

	require.Equal(t, "C", last.FromStop.ID)
	require.Equal(t, "A", last.ToStop.ID)

	require.True(t, last.IsLoopClosing)
	require.Greater(t, last.ToMeasure, last.FromMeasure)
	require.NotEmpty(t, last.Geometry)
}
```

---

## Versioning GitHub

Gunakan semantic versioning:

```txt
MAJOR.MINOR.PATCH
```

Contoh:

```txt
v0.1.0
v0.2.0
v1.0.0
v2.0.0
```

Aturan:

```txt
PATCH: bug fix
MINOR: fitur baru tanpa breaking change
MAJOR: breaking change
```

---

## Release Awal

```bash
git init
git add .
git commit -m "initial bus snap library"

git branch -M main
git remote add origin https://github.com/tije-syntra/snap-to-line.git
git push -u origin main

git tag -a v0.1.0 -m "initial experimental release"
git push origin v0.1.0
```

---

## Update Version

### Bug Fix

```bash
git checkout main
git pull origin main

git checkout -b fix/loop-stop-projection
# edit code

git add .
git commit -m "fix loop stop projection for same start end stop"

git checkout main
git merge fix/loop-stop-projection

git tag -a v0.1.1 -m "fix loop stop projection for same start end stop"
git push origin main
git push origin v0.1.1
```

---

### Tambah Fitur

```bash
git checkout develop
git pull origin develop

git checkout -b feature/direction-validation
# edit code

git add .
git commit -m "add direction validation for parallel route"

git checkout develop
git merge feature/direction-validation

git checkout main
git merge develop

git tag -a v0.2.0 -m "add direction validation for parallel route"
git push origin main develop
git push origin v0.2.0
```

---

### Breaking Change

```bash
git tag -a v1.0.0 -m "stable public API"
git push origin v1.0.0
```

Untuk Go module v2:

```go
module github.com/tije-syntra/snap-to-line/v2
```

Import:

```go
import "github.com/tije-syntra/snap-to-line/v2"
```

---

## Branch Maintenance

Gunakan branch:

```txt
main        -> stable release
develop     -> development aktif
feature/*   -> fitur baru
fix/*       -> bug fix
release/*   -> persiapan release
```

Flow:

```bash
git checkout develop
git pull origin develop

git checkout -b feature/viterbi-direction-loop
# coding

git add .
git commit -m "add viterbi direction loop handling"

git checkout develop
git merge feature/viterbi-direction-loop

git checkout main
git merge develop

git tag -a v0.3.0 -m "add viterbi direction loop handling"
git push origin main develop
git push origin v0.3.0
```

---

## Changelog

Buat file:

```txt
CHANGELOG.md
```

Contoh:

```md
# Changelog

## v0.3.0
- Add Viterbi direction validation
- Add trip direction scoring
- Add loop start-end stop handling
- Add measure-based segment slicing

## v0.2.0
- Add looping route support

## v0.1.0
- Initial release
- Add basic snap to linestring
- Add segment gate-to-gate
```

---

## Acceptance Criteria

Library siap jika:

* Bisa snap GPS ke linestring.
* Bisa membuat segment gate-to-gate.
* Bisa menangani rute reguler.
* Bisa menangani rute looping.
* Bisa menangani stop awal sama dengan stop akhir.
* Bisa membedakan occurrence stop yang ID dan koordinatnya sama.
* Bisa slice segment berdasarkan measure.
* Bisa validasi arah bus terhadap arah segment.
* Bisa membedakan jalur berangkat dan pulang yang sejajar.
* Bisa menjaga snapping stabil dengan Viterbi.
* Ada unit test untuk loop closure stop.
* Ada unit test untuk jalur sejajar.
* Ada unit test untuk bearing diff.
* Sudah memiliki tag minimal `v0.1.0`.

---

## Prioritas Implementasi

```txt
1. Geometry helper
2. Measure helper
3. Bearing helper
4. Sequential stop projection
5. Loop closure detection
6. Segment builder by measure
7. Basic snap to segment
8. Candidate search
9. Direction validation
10. Trip direction validation
11. Viterbi emission score
12. Viterbi transition score
13. Public API
14. Unit test
15. GitHub release
```

---

## Catatan Penting

Untuk rute looping:

```txt
Stop awal dan stop akhir boleh sama secara ID dan koordinat.
Tapi secara perjalanan, keduanya bukan stop yang sama.
Yang membedakan adalah occurrence dan measure.
```

Jadi:

```txt
A awal = occurrence 1, measure 0
A akhir = occurrence 2, measure totalLength
```

Segment harus dibuat dari:

```txt
measure ke measure
```

Bukan dari:

```txt
koordinat ke koordinat
```

Untuk jalur sejajar:

```txt
Candidate yang paling dekat belum tentu benar.
Candidate yang sedikit lebih jauh tetapi arahnya sesuai sering kali lebih benar.
```
