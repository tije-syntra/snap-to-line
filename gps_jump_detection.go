package snaptoline

const (
	DefaultGpsJumpWarningRatio       = 1.5
	DefaultGpsJumpSuspiciousRatio    = 3.0
	DefaultGpsJumpRejectRatio        = 5.0
	DefaultGpsJumpCountDistanceMeter = 150.0
)

type GpsJumpLevel string

const (
	GpsJumpLevelNormal     GpsJumpLevel = "normal"
	GpsJumpLevelWarning    GpsJumpLevel = "warning"
	GpsJumpLevelSuspicious GpsJumpLevel = "suspicious"
	GpsJumpLevelReject     GpsJumpLevel = "reject"
)

type GpsJumpEvaluation struct {
	Level            GpsJumpLevel
	GpsDistance      float64
	ExpectedDistance float64
	JumpRatio        float64
}

func resolveGpsJumpWarningRatio(cfg Config) float64 {
	if cfg.GpsJumpWarningRatio > 0 {
		return cfg.GpsJumpWarningRatio
	}
	return DefaultGpsJumpWarningRatio
}

func resolveGpsJumpSuspiciousRatio(cfg Config) float64 {
	if cfg.GpsJumpSuspiciousRatio > 0 {
		return cfg.GpsJumpSuspiciousRatio
	}
	return DefaultGpsJumpSuspiciousRatio
}

func resolveGpsJumpRejectRatio(cfg Config) float64 {
	if cfg.GpsJumpRejectRatio > 0 {
		return cfg.GpsJumpRejectRatio
	}
	return DefaultGpsJumpRejectRatio
}

func resolveGpsJumpCountDistanceMeter(cfg Config) float64 {
	if cfg.GpsJumpCountDistanceMeter > 0 {
		return cfg.GpsJumpCountDistanceMeter
	}
	return DefaultGpsJumpCountDistanceMeter
}

func ClassifyJumpRatio(jumpRatio float64, cfg Config) GpsJumpLevel {
	if jumpRatio < resolveGpsJumpWarningRatio(cfg) {
		return GpsJumpLevelNormal
	}
	if jumpRatio < resolveGpsJumpSuspiciousRatio(cfg) {
		return GpsJumpLevelWarning
	}
	if jumpRatio < resolveGpsJumpRejectRatio(cfg) {
		return GpsJumpLevelSuspicious
	}
	return GpsJumpLevelReject
}

func UpdateJumpCount(state *ViterbiState, gpsDistance float64, cfg Config) {
	if gpsDistance > resolveGpsJumpCountDistanceMeter(cfg) {
		state.JumpCount++
	}
}

func EvaluateGpsJump(state *ViterbiState, cfg Config, point GPSPoint) *GpsJumpEvaluation {
	if !cfg.GpsJumpDetection || state.LastPoint == nil {
		return nil
	}

	gpsDistance := DistanceMeter(state.LastPoint.Point, point.Point)
	expectedDistance := expectedGpsDistanceM(cfg, point, state.LastPoint, state.LastTimestamp)
	jumpRatio := 0.0
	if expectedDistance > 0 {
		jumpRatio = gpsDistance / expectedDistance
	}
	level := ClassifyJumpRatio(jumpRatio, cfg)

	UpdateJumpCount(state, gpsDistance, cfg)
	state.LastGpsJumpRatio = jumpRatio
	state.LastGpsJumpLevel = string(level)

	return &GpsJumpEvaluation{
		Level:            level,
		GpsDistance:      gpsDistance,
		ExpectedDistance: expectedDistance,
		JumpRatio:        jumpRatio,
	}
}

func (s *Snapper) evaluateGpsJump(point GPSPoint) *GpsJumpEvaluation {
	return EvaluateGpsJump(s.state, s.config, point)
}

func (s *Snapper) stabilizeGpsJump(point GPSPoint, evaluation *GpsJumpEvaluation) (*SnapResult, *Candidate) {
	if evaluation == nil || evaluation.Level != GpsJumpLevelReject {
		return nil, nil
	}
	ref := s.state.LastBest
	if ref == nil {
		ref = s.state.LastGood
	}
	if ref == nil {
		return nil, nil
	}
	return s.freezeAtRefCandidate(point, ref, "gps_jump_reject")
}

func ApplyGpsJumpResultMetadata(state *ViterbiState, cfg Config, result *SnapResult) *SnapResult {
	if !cfg.GpsJumpDetection || result == nil {
		return result
	}

	out := *result
	out.JumpCount = state.JumpCount
	if state.LastGpsJumpRatio > 0 {
		out.GpsJumpRatio = state.LastGpsJumpRatio
	}
	if state.LastGpsJumpLevel != "" {
		out.GpsJumpLevel = state.LastGpsJumpLevel
	}

	switch GpsJumpLevel(state.LastGpsJumpLevel) {
	case GpsJumpLevelWarning:
		out.Confidence = clampConfidence(out.Confidence * 0.9)
	case GpsJumpLevelSuspicious:
		out.Confidence = clampConfidence(out.Confidence * 0.75)
	}

	return &out
}
