package kit

import "math"

type pressureShapeResult struct {
	ZMin   float64
	ZMax   float64
	Curve  float64
	Scaled float64
}

func globalPressureShape(raw float64, domain string) pressureShapeResult {
	prefix := "global.pressure" + domain
	zmin := getGlobalPressureFloat(prefix+"zmin", 0.0)
	zmax := getGlobalPressureFloat(prefix+"zmax", 1.0)
	curve := getGlobalPressureFloat(prefix+"curve", 1.0)
	if curve <= 0.0 {
		curve = 1.0
	}

	scaled := scalePressureRange(raw, zmin, zmax)
	scaled = math.Pow(scaled, curve)
	return pressureShapeResult{
		ZMin:   zmin,
		ZMax:   zmax,
		Curve:  curve,
		Scaled: boundValueZeroToOne(scaled),
	}
}

func getGlobalPressureFloat(name string, dflt float64) float64 {
	if GlobalParams == nil {
		return dflt
	}
	value, err := GetParamFloat(name)
	if err != nil {
		return dflt
	}
	return value
}

func scalePressureRange(raw, zmin, zmax float64) float64 {
	if zmax <= zmin {
		return boundValueZeroToOne(raw)
	}
	return boundValueZeroToOne(BoundAndScaleFloat(raw, zmin, zmax, 0.0, 1.0))
}

func pressureToVelocity(scaledPressure float64, velocitymin, velocitymax int) uint8 {
	if velocitymin > velocitymax {
		velocitymin, velocitymax = velocitymax, velocitymin
	}
	velocity := velocitymin + int(math.Round(boundValueZeroToOne(scaledPressure)*float64(velocitymax-velocitymin)))
	if velocity < 0 {
		velocity = 0
	}
	if velocity > 127 {
		velocity = 127
	}
	return uint8(velocity)
}
