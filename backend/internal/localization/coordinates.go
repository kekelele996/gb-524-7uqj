package localization

import "math"

type Frame struct {
	OriginLatDeg float64 `json:"origin_latitude"`
	OriginLonDeg float64 `json:"origin_longitude"`
	cosLatitude  float64
}

func NewFrame(inputs []Input) Frame {
	var latitude, longitude float64
	for _, input := range inputs {
		latitude += input.Latitude
		longitude += input.Longitude
	}
	if len(inputs) > 0 {
		latitude /= float64(len(inputs))
		longitude /= float64(len(inputs))
	}
	return Frame{
		OriginLatDeg: latitude,
		OriginLonDeg: longitude,
		cosLatitude:  math.Cos(degreesToRadians(latitude)),
	}
}

func (f Frame) ToLocal(latitude, longitude float64) Point {
	latDelta := degreesToRadians(latitude - f.OriginLatDeg)
	lonDelta := degreesToRadians(longitude - f.OriginLonDeg)
	return Point{
		X: EarthRadiusM * lonDelta * f.cosLatitude,
		Y: EarthRadiusM * latDelta,
	}
}

func (f Frame) ToGeo(point Point) GeoPoint {
	latitude := f.OriginLatDeg + radiansToDegrees(point.Y/EarthRadiusM)
	longitude := f.OriginLonDeg
	if math.Abs(f.cosLatitude) > 1e-12 {
		longitude += radiansToDegrees(point.X / (EarthRadiusM * f.cosLatitude))
	}
	return GeoPoint{Latitude: latitude, Longitude: longitude}
}

func BearingDirection(degrees float64) Point {
	radians := degreesToRadians(normalizeBearing(degrees))
	return Point{X: math.Sin(radians), Y: math.Cos(radians)}
}

func BearingFromVector(deltaX, deltaY float64) float64 {
	return normalizeBearing(radiansToDegrees(math.Atan2(deltaX, deltaY)))
}

func normalizeBearing(value float64) float64 {
	value = math.Mod(value, 360)
	if value < 0 {
		value += 360
	}
	return value
}

func normalizeAngleDelta(value float64) float64 {
	value = math.Mod(value+180, 360)
	if value < 0 {
		value += 360
	}
	return value - 180
}

func degreesToRadians(value float64) float64 { return value * math.Pi / 180 }
func radiansToDegrees(value float64) float64 { return value * 180 / math.Pi }
