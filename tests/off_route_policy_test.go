package snaptoline_test

import (
	"testing"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func testOffRoutePolicy(maxSnap float64) snaptoline.OffRoutePolicy {
	return snaptoline.OffRoutePolicy{
		MaxSnapDistanceMeter:     maxSnap,
		OffRouteSoftDistFraction: 0.54,
		OffRouteMinConfidence:    0.2,
		OffRouteSoftConfidence:   0.5,
	}
}

func TestMapOffRoute_respectsMaxSnapSetting(t *testing.T) {
	policy := testOffRoutePolicy(100)
	result := &snaptoline.SnapResult{
		DistanceMeter: 38,
		Confidence:    0.85,
		IsOffRoute:    false,
	}
	if snaptoline.MapOffRoute(result, false, policy) {
		t.Fatal("38m must not be off-route when max snap is 100m")
	}
	off28 := &snaptoline.SnapResult{
		DistanceMeter: 38,
		Confidence:    0.85,
		IsOffRoute:    true,
	}
	if !snaptoline.MapOffRoute(off28, false, testOffRoutePolicy(28)) {
		t.Fatal("38m should be off-route when max snap is 28m")
	}
}

func TestMapOffRoute_oldHardcoded22WouldHaveFlagged(t *testing.T) {
	result := &snaptoline.SnapResult{
		DistanceMeter: 38,
		Confidence:    0.9,
		IsOffRoute:    false,
	}
	if snaptoline.MapOffRoute(result, false, testOffRoutePolicy(100)) {
		t.Fatal("must not use legacy 22m off-route cutoff")
	}
}

func TestMapOffRoute_heldSegmentMasks(t *testing.T) {
	result := &snaptoline.SnapResult{
		DistanceMeter: 50,
		Confidence:    0.8,
		IsOffRoute:    true,
		HeldSegment:   true,
	}
	if snaptoline.MapOffRoute(result, false, testOffRoutePolicy(28)) {
		t.Fatal("held segment should mask off-route on map")
	}
}

func TestMapOffRoute_lowConfidence(t *testing.T) {
	result := &snaptoline.SnapResult{
		DistanceMeter: 8,
		Confidence:    0.1,
		IsOffRoute:    false,
	}
	if !snaptoline.MapOffRoute(result, false, testOffRoutePolicy(100)) {
		t.Fatal("very low confidence should still show off-route")
	}
}

func TestMapOffRoute_customSoftThreshold(t *testing.T) {
	policy := snaptoline.OffRoutePolicy{
		MaxSnapDistanceMeter:     100,
		OffRouteSoftDistFraction: 0.5,
		OffRouteMinConfidence:    0.2,
		OffRouteSoftConfidence:   0.6,
	}
	result := &snaptoline.SnapResult{
		DistanceMeter: 55,
		Confidence:    0.55,
		IsOffRoute:    false,
	}
	if !snaptoline.MapOffRoute(result, false, policy) {
		t.Fatal("expected soft off-route with custom policy")
	}
}

func TestEtaSnapReliableForPublish_heldSegmentFarFromLine(t *testing.T) {
	result := &snaptoline.SnapResult{
		SegmentOrder:  3,
		DistanceMeter: 35,
		HeldSegment:   true,
		IsOffRoute:    false,
		Confidence:    0.8,
	}
	policy := testOffRoutePolicy(28)
	if snaptoline.EtaSnapReliableForPublish(result, false, policy) {
		t.Fatal("held segment with GPS far from line must not refresh ETA/PIS")
	}
}

func TestEtaSnapReliableForPublish_onRoute(t *testing.T) {
	result := &snaptoline.SnapResult{
		SegmentOrder:  2,
		DistanceMeter: 8,
		Confidence:    0.9,
	}
	policy := testOffRoutePolicy(28)
	if !snaptoline.EtaSnapReliableForPublish(result, false, policy) {
		t.Fatal("expected reliable snap on route")
	}
}

func TestEtaSnapReliableForPublish_offRoute(t *testing.T) {
	result := &snaptoline.SnapResult{
		SegmentOrder:  2,
		DistanceMeter: 5,
		IsOffRoute:    true,
		Confidence:    0.9,
	}
	policy := testOffRoutePolicy(28)
	if snaptoline.EtaSnapReliableForPublish(result, false, policy) {
		t.Fatal("off-route snap must not refresh ETA/PIS")
	}
}

func TestSnapDegraded_heldOffRoute(t *testing.T) {
	result := &snaptoline.SnapResult{
		SegmentID:    "SEG-A-B-1",
		SegmentOrder: 1,
		HeldSegment:  true,
		IsOffRoute:   true,
	}
	if !snaptoline.SnapDegraded(result) {
		t.Fatal("held + off-route should be degraded")
	}
}
