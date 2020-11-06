package stat

import "math"

// A TDist is a Student's t-distribution with V degrees of freedom.
type TDist struct {
	V float64
}

// PDF is the probability distribution function
func (t TDist) PDF(x float64) float64 {
	return math.Exp(lgamma((t.V+1)/2)-lgamma(t.V/2)) /
		math.Sqrt(t.V*math.Pi) * math.Pow(1+(x*x)/t.V, -(t.V+1)/2)
}

// CDF is the cumulative distribution function
func (t TDist) CDF(x float64) float64 {
	if x == 0 {
		return 0.5
	} else if x > 0 {
		return 1 - 0.5*mathBetaInc(t.V/(t.V+x*x), t.V/2, 0.5)
	} else if x < 0 {
		return 1 - t.CDF(-x)
	} else {
		return math.NaN()
	}
}

// Bounds ...
func (t TDist) Bounds() (float64, float64) {
	return -4, 4
}
