package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func foldedSegmentLine() orb.LineString {
	return orb.LineString{
		{106.656100, -6.129000},
		{106.656250, -6.129150},
		{106.656260, -6.129180},
		{106.656310, -6.129220},
		{106.656350, -6.129280},
		{106.656200, -6.129100},
	}
}

func foldedSegmentStops() []snaptoline.Stop {
	line := foldedSegmentLine()
	return []snaptoline.Stop{
		{ID: "S5", Order: 1, Point: line[0]},
		{ID: "S6", Order: 2, Point: line[1]},
		{ID: "B08326P", Order: 3, Point: line[4]},
		{ID: "S8", Order: 4, Point: line[5]},
	}
}

func TestFoldedSegmentBranchLockPicksNearestAndStays(t *testing.T) {
	line := foldedSegmentLine()
	stops := foldedSegmentStops()
	cfg := snaptoline.RouteSnapConfig(stops)
	require.True(t, cfg.FoldedSegmentBranchLock)

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	_, err = snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.656310, -6.129220}, Bearing: 137, Speed: 11})
	require.NoError(t, err)
	south, err := snapper.Snap(snaptoline.GPSPoint{Point: orb.Point{106.656260, -6.129180}, Bearing: 137, Speed: 11})
	require.NoError(t, err)
	require.Equal(t, 2, south.SegmentOrder)

	noisy := []orb.Point{
		{106.656327, -6.129180},
		{106.656320, -6.129175},
		{106.656335, -6.129182},
		{106.656318, -6.129178},
		{106.656330, -6.129181},
	}
	var prevSnap orb.Point
	var havePrev bool
	for _, p := range noisy {
		r, err := snapper.Snap(snaptoline.GPSPoint{Point: p, Bearing: 96, Speed: 0})
		require.NoError(t, err)
		require.Equal(t, 2, r.SegmentOrder)
		if havePrev {
			jump := snaptoline.DistanceMeter(prevSnap, r.SnappedPoint)
			require.Less(t, jump, 4.0, "branch lock should prevent flip-flop between parallel branches")
		}
		prevSnap = r.SnappedPoint
		havePrev = true
	}
}

func TestFoldedSegmentBranchLockEnabledByDefault(t *testing.T) {
	cfg := snaptoline.RouteSnapConfig(foldedSegmentStops())
	require.True(t, cfg.FoldedSegmentBranchLock)
	require.GreaterOrEqual(t, cfg.FoldedSegmentMinViable, 3)
}
