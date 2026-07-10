package snaptoline_test

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"

	snaptoline "github.com/tije-syntra/snap-to-line"
)

func TestIsGpsTeleportDetectsFastMovement(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.TeleportDetection = true
	cfg.TeleportDistanceMeter = 300
	cfg.TeleportTimeSec = 5

	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)
	state.LastPoint = &snaptoline.GPSPoint{
		Point:     orb.Point{106.0, -6.0},
		Speed:     5,
		Timestamp: 1000,
	}
	state.LastTimestamp = 1000

	teleport := snaptoline.IsGpsTeleport(state, cfg, snaptoline.GPSPoint{
		Point:     orb.Point{106.03, -6.0},
		Speed:     5,
		Timestamp: 4000,
	})
	require.True(t, teleport)
}

func TestIsGpsTeleportSkipsWhenSpeedMatches(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.TeleportDetection = true
	cfg.TeleportDistanceMeter = 300
	cfg.TeleportTimeSec = 5

	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)
	state.LastPoint = &snaptoline.GPSPoint{
		Point:     orb.Point{106.0, -6.0},
		Speed:     250,
		Timestamp: 1000,
	}
	state.LastTimestamp = 1000

	teleport := snaptoline.IsGpsTeleport(state, cfg, snaptoline.GPSPoint{
		Point:     orb.Point{106.003, -6.0},
		Speed:     250,
		Timestamp: 6000,
	})
	require.False(t, teleport)
}

func TestIsGpsTeleportSkipsOutsideTimeWindow(t *testing.T) {
	cfg := snaptoline.DefaultConfig()
	cfg.TeleportDetection = true
	cfg.TeleportTimeSec = 5

	state := snaptoline.NewViterbiState(snaptoline.DirectionUnknown)
	state.LastPoint = &snaptoline.GPSPoint{
		Point:     orb.Point{106.0, -6.0},
		Speed:     5,
		Timestamp: 1000,
	}
	state.LastTimestamp = 1000

	teleport := snaptoline.IsGpsTeleport(state, cfg, snaptoline.GPSPoint{
		Point:     orb.Point{106.03, -6.0},
		Speed:     5,
		Timestamp: 8000,
	})
	require.False(t, teleport)
}

func TestSnapperHoldsPositionOnTeleport(t *testing.T) {
	line := orb.LineString{{106.0, -6.0}, {106.1, -6.0}}
	stops := []snaptoline.Stop{
		{ID: "A", Order: 1, Point: line[0]},
		{ID: "B", Order: 2, Point: line[1]},
	}

	cfg := snaptoline.DefaultConfig()
	cfg.TeleportDetection = true
	cfg.TeleportDistanceMeter = 300
	cfg.TeleportTimeSec = 5

	snapper, err := snaptoline.NewSnapper(line, stops, cfg)
	require.NoError(t, err)

	ts := int64(1000)
	first, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.01, -6.0001},
		Speed:     30,
		Timestamp: ts,
	})
	require.NoError(t, err)
	require.Greater(t, first.SegmentOrder, 0)

	ts += 3000
	teleported, err := snapper.Snap(snaptoline.GPSPoint{
		Point:     orb.Point{106.05, -6.0001},
		Speed:     5,
		Timestamp: ts,
	})
	require.NoError(t, err)
	require.True(t, teleported.HeldSegment)
	require.Equal(t, "teleport_detected", teleported.HeldReason)
	require.Equal(t, first.SegmentOrder, teleported.SegmentOrder)
}
