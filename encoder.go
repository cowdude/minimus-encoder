package minimus

import (
	"io"
	"math"
	"math/bits"
	"unsafe"

	"github.com/icza/bitio"
)

type elementState struct {
	prevBits    uint64
	lead, trail uint8
}

type Encoder struct {
	writer *bitio.Writer
	state  []elementState
	first  bool
}

const (
	shiftSize        = 6
	numValueBitsSize = 6
	eofShiftBits     = (1 << shiftSize) - 1
	eofNumValueBits  = (1 << numValueBitsSize) - 1
	floatBitsSize    = uint8(unsafe.Sizeof(float64(0)) * 8)
)

func LossyFloat64(n float64, maxAbsError float64) float64 {
	bits := math.Float64bits(n)
	const bitmask uint64 = (1 << 64) - 1
	const maxBits = 52

	//binary search for most trailing bits to zero-out
	var (
		i uint8
		j uint8 = maxBits - 1
	)
	for i < j {
		h := (i + j) >> 1 //no possible overflow
		np := math.Float64frombits(bits & (bitmask << (h + 1)))

		// i ≤ h < j
		if math.Abs(np-n) < maxAbsError {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	return math.Float64frombits(bits & (bitmask << i))
}

func NewEncoder(dst io.Writer, span int) *Encoder {
	state := make([]elementState, span)
	for i := range state {
		state[i] = elementState{lead: 0xFF}
	}
	return &Encoder{
		writer: bitio.NewWriter(dst),
		first:  true,
		state:  state,
	}
}

func (enc *Encoder) Put(vec Vec64) error {
	if len(vec) != len(enc.state) {
		panic("Wrong vector size")
	}
	if enc.first {
		for i, bits := range vec {
			enc.writer.TryWriteBits(bits, floatBitsSize)
			enc.state[i].prevBits = bits
		}
		enc.first = false
		return enc.writer.TryError
	}

	for i, xbits := range vec {
		diff := xbits ^ enc.state[i].prevBits
		ctl0 := diff == 0
		enc.writer.TryWriteBool(ctl0)
		if ctl0 {
			continue
		}

		leading := uint8(bits.LeadingZeros64(diff))
		trailing := uint8(bits.TrailingZeros64(diff))
		numValueBits := floatBitsSize - enc.state[i].lead - enc.state[i].trail
		newNumValueBits := floatBitsSize - leading - trailing
		sizeNewWin := shiftSize + numValueBitsSize + newNumValueBits
		ctl1 := trailing >= enc.state[i].trail &&
			leading >= enc.state[i].lead &&
			numValueBits <= sizeNewWin
		enc.writer.TryWriteBool(ctl1)

		if ctl1 {
			enc.writer.TryWriteBits(diff>>enc.state[i].trail, numValueBits)
			enc.state[i].prevBits = xbits
			continue
		}

		encodedNumValueBits := newNumValueBits - 1
		if trailing > eofShiftBits {
			panic("trailing overflow")
		}
		if encodedNumValueBits > eofNumValueBits {
			panic("encodedNumValueBits overflow")
		}
		if trailing == eofShiftBits && encodedNumValueBits == eofNumValueBits {
			panic("invalid trail/value bits: overlaps with eof")
		}
		window := (uint64(trailing) << numValueBitsSize) | uint64(encodedNumValueBits)
		enc.writer.TryWriteBits(window, shiftSize+numValueBitsSize)
		enc.writer.TryWriteBits(diff>>trailing, newNumValueBits)
		enc.state[i].lead = leading
		enc.state[i].trail = trailing
		enc.state[i].prevBits = xbits
	}

	return enc.writer.TryError
}

func (enc *Encoder) PutFloat64(vec []float64) {
	sh := (*Vec64)(unsafe.Pointer(&vec))
	enc.Put(*sh)
}

func (enc *Encoder) PutUint64(vec []uint64) {
	enc.Put([]uint64(vec))
}

func (enc *Encoder) Close() error {
	const numControlBits = 2
	const eofBits = eofNumValueBits | (eofShiftBits << numValueBitsSize)
	enc.writer.TryWriteBits(eofBits, numControlBits+shiftSize+numValueBitsSize)
	return enc.writer.Close()
}
