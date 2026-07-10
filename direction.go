package snaptoline

import "math"

func DirectionScore(diff, maxDiff float64) float64 {
	if maxDiff <= 0 {
		return 1
	}
	if diff > maxDiff {
		return 0.05
	}
	return 1 - (diff / maxDiff)
}

func TripDirectionScore(candidate, active DirectionType) float64 {
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

func EmissionScore(distanceMeter, maxDistance float64) float64 {
	if maxDistance <= 0 {
		return 0.05
	}
	if distanceMeter > maxDistance {
		return 0.05
	}
	ratio := distanceMeter / maxDistance
	score := 1 - ratio
	if score < 0.05 {
		return 0.05
	}
	return score
}

func resolveBusBearing(point GPSPoint, prev *GPSPoint, cfg Config) (float64, bool) {
	if point.Bearing > 0 {
		return point.Bearing, true
	}

	if prev == nil || !cfg.UseMovementBearing {
		return 0, false
	}

	movement := DistanceMeter(prev.Point, point.Point)
	if movement < cfg.MinMovementMeter {
		return 0, false
	}

	return BearingBetween(prev.Point, point.Point), true
}

func ShouldWeakenDirectionValidation(point GPSPoint, prev *GPSPoint, cfg Config, turnaroundValidated bool) bool {
	return shouldWeakenDirectionValidation(point, prev, cfg, turnaroundValidated)
}

func shouldWeakenDirectionValidation(point GPSPoint, prev *GPSPoint, cfg Config, turnaroundValidated bool) bool {
	if turnaroundValidated && cfg.ReverseDetection {
		return true
	}

	const lowSpeedKmh = 3.0

	// Speed is km/h (live GPS feeds). Treat stopped / crawl as dwell.
	if cfg.UseSpeed && point.Speed < lowSpeedKmh {
		return true
	}

	if prev != nil && cfg.UseMovementBearing {
		movement := DistanceMeter(prev.Point, point.Point)
		if movement < cfg.MinMovementMeter {
			return true
		}
	}

	return false
}

func scoreDirection(busBearing float64, hasBearing bool, lineBearing float64, cfg Config, weaken bool) (float64, float64) {
	if !cfg.UseBearingValidation || !hasBearing {
		return 1, 0
	}

	diff := BearingDiff(busBearing, lineBearing)
	score := DirectionScore(diff, cfg.MaxBearingDiffDegree)
	if weaken && score < 0.7 {
		score = 0.7
	}
	return score, diff
}

func logScore(score float64) float64 {
	if score <= 0 {
		return math.Log(0.05)
	}
	return math.Log(score)
}
