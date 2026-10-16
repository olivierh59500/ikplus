package graphics

import "math"

// GroundShadow places an authored eight-row cell on the scene's ground strip.
type GroundShadow struct {
	Left, Shape, Shift, Edge int
}

// ProjectileGroundShadow uses projectile centers in the native scene. Ground
// cells follow whole horizontal positions, independently of sprite fractions.
func ProjectileGroundShadow(x, y float64, floor int) (GroundShadow, bool) {
	height := floor - int(y)
	if height < 0 || height >= 8 {
		return GroundShadow{}, false
	}
	rawX := int(math.Floor((x+24)/2)) & 255
	if rawX < 8 || rawX >= 176 {
		return GroundShadow{}, false
	}
	edge := 0
	if rawX < 16 {
		edge = 1
	} else if rawX >= 168 {
		edge = 2
	}
	return GroundShadow{Left: (rawX&248)*2 - 32, Shape: height, Shift: (rawX & 7) * 2, Edge: edge}, true
}
