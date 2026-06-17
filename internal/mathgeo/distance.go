package mathgeo

import (
	"math"

	"github.com/paulmach/orb"
)

const earthRadiusMeter = 6371008.8

func DistanceMeter(a, b orb.Point) float64 {
	lat1 := toRad(a[1])
	lat2 := toRad(b[1])
	dLat := toRad(b[1] - a[1])
	dLon := toRad(b[0] - a[0])

	sinDLat := math.Sin(dLat / 2)
	sinDLon := math.Sin(dLon / 2)

	h := sinDLat*sinDLat + math.Cos(lat1)*math.Cos(lat2)*sinDLon*sinDLon
	return 2 * earthRadiusMeter * math.Asin(math.Min(1, math.Sqrt(h)))
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}

func toDeg(rad float64) float64 {
	return rad * 180 / math.Pi
}
