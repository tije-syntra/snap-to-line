package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestLiveBusSnapConfig_nonLoop(t *testing.T) {
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: orb.Point{106.0, -6.0}},
		{ID: "B", Order: 2, Point: orb.Point{106.05, -6.0}},
		{ID: "C", Order: 3, Point: orb.Point{106.1, -6.0}},
	}
	cfg := snaptoline.LiveBusSnapConfig(stops)
	if cfg.MaxSnapDistanceMeter != snaptoline.RecommendedMaxSnapDistanceMeter {
		t.Fatalf("MaxSnapDistanceMeter = %v", cfg.MaxSnapDistanceMeter)
	}
	if cfg.MeasureRegressionToleranceMeter != snaptoline.RecommendedMeasureRegressionToleranceMeter {
		t.Fatalf("MeasureRegressionToleranceMeter = %v", cfg.MeasureRegressionToleranceMeter)
	}
	if !cfg.SnapDistanceResetOnGrow {
		t.Fatal("expected grow reset on non-loop routes")
	}
	if cfg.SnapDistanceResetMaxMeter != snaptoline.RecommendedSnapDistanceResetMaxMeter {
		t.Fatalf("SnapDistanceResetMaxMeter = %v", cfg.SnapDistanceResetMaxMeter)
	}
	if !cfg.RequireStopRadiusForSegmentSwitch {
		t.Fatal("expected stop-radius segment switch")
	}
}

func TestLiveBusSnapConfig_loop(t *testing.T) {
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: orb.Point{106.0, -6.0}},
		{ID: "B", Order: 2, Point: orb.Point{106.05, -6.0}},
		{ID: "A", Order: 3, Point: orb.Point{106.0, -6.0}},
	}
	if !snaptoline.IsLoopRoute(stops) {
		t.Fatal("expected loop route")
	}
	cfg := snaptoline.LiveBusSnapConfig(stops)
	if cfg.SnapDistanceResetOnGrow {
		t.Fatal("loop routes should disable grow reset")
	}
	if cfg.NextStopPassToleranceMeter != snaptoline.RecommendedLoopNextStopPassToleranceMeter {
		t.Fatalf("NextStopPassToleranceMeter = %v", cfg.NextStopPassToleranceMeter)
	}
}

func TestDefaultOffRoutePolicy_matchesLiveBus(t *testing.T) {
	p := snaptoline.DefaultOffRoutePolicy()
	if p.MaxSnapDistanceMeter != snaptoline.RecommendedMaxSnapDistanceMeter {
		t.Fatalf("off-route max snap = %v", p.MaxSnapDistanceMeter)
	}
}
