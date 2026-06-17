package mathgeo

import (
	"math"

	"github.com/paulmach/orb"
)

func BearingBetween(a, b orb.Point) float64 {
	lat1 := toRad(a[1])
	lat2 := toRad(b[1])
	dLon := toRad(b[0] - a[0])

	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)

	bearing := toDeg(math.Atan2(y, x))
	if bearing < 0 {
		bearing += 360
	}
	return bearing
}

func BearingDiff(a, b float64) float64 {
	diff := math.Abs(a - b)
	if diff > 180 {
		diff = 360 - diff
	}
	return diff
}
