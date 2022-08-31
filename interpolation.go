package edau

// An interpolator function receives a slice of values, the position at which
// we want to interpolate, and returns the interpolated value.
//
// Positions to interpolate must always be within 0 <= position <= len(samples) - 1.
// Responsible callers will try to keep it as close to (len(samples) - 1)/2 as
// possible.
//
// Interpolation functions are mainly used for resampling processes.
type InterpolatorFunc func([]float64, float64) float64

// Credit to Olli's paper (http://yehar.com/blog/wp-content/uploads/2009/08/deip.pdf),
// most code in this file is simply following that paper.

// Lagrange interpolation for N samples. Samples are considered to start at zero,
// and targetPosition must be an index near the middle of samples' slice length
// for best results. This function is slow; for faster interpolation use one of
// the fixed-N functions instead.
func InterpLagrangeN(samples []float64, targetPosition float64) float64 {
	output := 0.0
	for j, sample := range samples {
		// compute lagrange basis polynomial for the current sample
		l := 1.0
		for m, _ := range samples {
			if j == m { continue }
			l *= (targetPosition - float64(m))/float64(j - m)
		}

		// apply contribution of the basis polynomial to the output
		// >> L(targetPosition) = sum(sample_j*l_j) for every j
		output += sample*l
	}

	return output
}

// 4-point, 3rd-order Lagrange interpolation. Samples are considered to start at zero.
// x is the position at which we want to interpolate. For best results, x must be
// reasonably close to len(samples)/2.
func InterpLagrange4Pt3Ord(samples []float64, x float64) float64 {
	c0 := samples[1]
	c1 := samples[2] - 1.0/3.0*samples[0] - 0.5*samples[1] - 1.0/6.0*samples[3]
	c2 := 0.5*(samples[0] + samples[2]) - samples[1]
	c3 := 1.0/6.0*(samples[3] - samples[0]) + 0.5*(samples[1] - samples[2])
	x1 := x - 1.0
	return ((c3*x1 + c2)*x1 + c1)*x1 + c0

	// non-optimized polynomial version
	// 	l0 := -(x - 1)*((x - 2)/2)*((x - 3)/3)
	// 	l1 := x*(x - 2)*((x - 3)/2)
	// 	l2 := (x/2)*(x - 1)*-(x - 3)
	// 	l3 := (x/3)*((x - 1)/2)*(x - 2)
	// 	return samples[0]*l0 + samples[1]*l1 + samples[2]*l2 + samples[3]*l3
}

// 6-point, 5th-order Lagrange interpolation. Samples are considered to start at zero.
// x is the position at which we want to interpolate. For best results, x must be
// reasonably close to len(samples)/2.
func InterpLagrange6Pt5Ord(samples []float64, x float64) float64 {
	// precomputed values for some recurrent matrix values
	sam_1_3_pre := samples[1] + samples[3]
	sam_0_4_pre := 1/24.0*(samples[0] + samples[4])

	// lagrange polynomial resolution
	c0 := samples[2]
	c1 := 1.0/20.0*samples[0] - 0.5*samples[1] - 1.0/3.0*samples[2] + samples[3] - 0.25*samples[4] + 1.0/30.0*samples[5]
	c2 := 2.0/3.0*sam_1_3_pre - 5.0/4.0*samples[2] - sam_0_4_pre
	c3 := 5.0/12.0*samples[2] - 7.0/12.0*samples[3] + 7.0/24.0*samples[4] - 1.0/24.0*(samples[0] + samples[1] + samples[5])
	c4 := 0.25*samples[2] - 1.0/6.0*sam_1_3_pre + sam_0_4_pre
	c5 := 1.0/120.0*(samples[5] - samples[0]) + 1/24.0*(samples[1] - samples[4]) + 1.0/12.0*(samples[3] - samples[2])

	x2 := x - 2.0
	return ((((c5*x2 + c4)*x2 + c3)*x2 + c2)*x2 + c1)*x2 + c0
}

// 4-point, 3rd-order Hermite interpolation. Samples are considered to start at zero.
// x is the position at which we want to interpolate. For best results, x must be
// reasonably close to len(samples)/2.
func InterpHermite4Pt3Ord(samples []float64, x float64) float64 {
	c0 := samples[1]
	c1 := 0.5*(samples[2] - samples[0])
	c2 := samples[0] - 2.5*samples[1] + 2.0*samples[2] - 0.5*samples[3]
	c3 := 0.5*(samples[3] - samples[0]) + 1.5*(samples[1] - samples[2])
	x1 := x - 1.0
	return ((c3*x1 + c2)*x1 + c1)*x1 + c0
}

// 6-point, 3rd-order Hermite interpolation. Samples are considered to start at zero.
// x is the position at which we want to interpolate. For best results, x must be
// reasonably close to len(samples)/2.
func InterpHermite6Pt3Ord(samples []float64, x float64) float64 {
	c0 := samples[2]
	c1 := 1.0/12.0*(samples[0] - samples[4]) + 2.0/3.0*(samples[3] - samples[1])
	c2 := 1.25*samples[1] - 7.0/3.0*samples[2] + 5.0/3.0*samples[3] - 0.5*samples[4] + 1.0/12.0*samples[5] - 1.0/6.0*samples[0]
	c3 := 1.0/12.0*(samples[0] - samples[5]) + 7.0/12.0*(samples[4] - samples[1]) + 4.0/3.0*(samples[2] - samples[3])
	x2 := x - 2.0
	return ((c3*x2 + c2)*x2 + c1)*x2 + c0
}

// This doesn't sound good for resampling so I left it commented. It's
// possible the code was written with some typo in the original paper.
// func InterpHermite6Pt5Ord(samples []float64, x float64) float64 {
// 	// precomputed values for some recurrent matrix values
// 	sam0_pre := 1.0/8.0*samples[0]
// 	sam4_pre := 11.0/24.0*samples[4]
// 	sam5_pre := 1.0/12.0*samples[5]

// 	// hermite polynomial resolution
// 	c0 := samples[2]
// 	c1 := 1.0/12.0*(samples[0] - samples[4]) + 1.5*(samples[3] - samples[1])
// 	c2 := 13.0/12.0*samples[1] - 25.0/12.0*samples[2] + 1.5*samples[3] - sam4_pre + sam5_pre - sam0_pre
// 	c3 := 5.0/12.0*samples[2] - 7.0/12.0*samples[3] + 7.0/24.0*samples[4] - 1.0/24.0*(samples[0] + samples[1] + samples[5])
// 	c4 := sam0_pre - 7.0/12.0*samples[1] + 13.0/12.0*samples[2] - samples[3] + sam4_pre - sam5_pre
// 	c5 := 1.0/24.0*(samples[5] - samples[0]) + 5.0/24.0*(samples[1] - samples[4]) + 5.0/12.0*(samples[3] - samples[2])
// 	x2 := x - 2.0
// 	return ((((c5*x2 + c4)*x2 + c3)*x2 + c2)*x2 + c1)*x2 + c0
// }
