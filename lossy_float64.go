package minimus

import "math"

var (
	leadingBitmasks [floatBitsSize]uint64
)

func init() {
	var bits uint64 = (1 << 64) - 1
	for i := range leadingBitmasks {
		leadingBitmasks[i] = bits
		bits <<= 1
	}
}

/*
LossyFloat64 transforms a float64 into a compress-friendly approximation.

The function guarantees that abs(result - n) < maxAbsError.

Under the hood, the function zero-outs as many least significant bits.
*/
func LossyFloat64(n float64, maxAbsError float64) float64 {
	bits := math.Float64bits(n)
	const maxBits = 52 //msb are used for sign and NAN => keep them

	//binary search for most trailing bits to zero-out
	var (
		i uint8
		j uint8 = maxBits - 1
	)
	for i < j {
		h := (i + j) >> 1 //no possible overflow
		np := math.Float64frombits(bits & leadingBitmasks[h+1])

		// i ≤ h < j
		if math.Abs(np-n) < maxAbsError {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	return math.Float64frombits(bits & leadingBitmasks[i])
}
