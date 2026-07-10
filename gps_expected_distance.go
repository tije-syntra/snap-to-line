package snaptoline

const (
	DefaultGpsJumpExpectedFactor   = 1.5
	DefaultGpsJumpMinExpectedMeter = 75.0
	DefaultGpsJumpFallbackSpeedKmh = 40.0
)

func resolveGpsJumpExpectedFactor(cfg Config) float64 {
	if cfg.GpsJumpExpectedFactor > 0 {
		return cfg.GpsJumpExpectedFactor
	}
	return DefaultGpsJumpExpectedFactor
}

func resolveGpsJumpMinExpectedMeter(cfg Config) float64 {
	if cfg.GpsJumpMinExpectedMeter > 0 {
		return cfg.GpsJumpMinExpectedMeter
	}
	return DefaultGpsJumpMinExpectedMeter
}

func resolveDeltaSec(point GPSPoint, lastTimestamp int64) float64 {
	nowTs := resolveTimestamp(point, 0)
	if nowTs <= 0 {
		nowTs = lastTimestamp
	}
	if lastTimestamp <= 0 {
		return 1
	}
	deltaSec := float64(nowTs-lastTimestamp) / 1000.0
	if deltaSec < 0.3 {
		deltaSec = 0.3
	}
	if deltaSec > 30 {
		deltaSec = 30
	}
	return deltaSec
}

func resolveSpeedKmhForJump(point GPSPoint, lastPoint *GPSPoint) float64 {
	speedKmh := point.Speed
	if speedKmh <= 0 && lastPoint != nil {
		speedKmh = lastPoint.Speed
	}
	if speedKmh <= 0 {
		speedKmh = DefaultGpsJumpFallbackSpeedKmh
	}
	return speedKmh
}

func ResolveGpsJumpMinExpectedMeter(cfg Config) float64 {
	return resolveGpsJumpMinExpectedMeter(cfg)
}

func ExpectedGpsDistanceM(cfg Config, point GPSPoint, lastPoint *GPSPoint, lastTimestamp int64) float64 {
	return expectedGpsDistanceM(cfg, point, lastPoint, lastTimestamp)
}

func expectedGpsDistanceM(cfg Config, point GPSPoint, lastPoint *GPSPoint, lastTimestamp int64) float64 {
	deltaSec := resolveDeltaSec(point, lastTimestamp)
	speedKmh := resolveSpeedKmhForJump(point, lastPoint)
	factor := resolveGpsJumpExpectedFactor(cfg)
	minExpected := resolveGpsJumpMinExpectedMeter(cfg)
	expected := (speedKmh / 3.6) * deltaSec * factor
	if expected < minExpected {
		return minExpected
	}
	return expected
}
