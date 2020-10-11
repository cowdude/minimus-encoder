package minimus

import (
	"io"
	"math/bits"
	"unsafe"

	"github.com/icza/bitio"
)

type elementState struct {
	prevBits    uint64
	lead, trail uint8
}

//Encoder stores the encoding context for compressing a sequence of
//Vec64 to an io.Writer
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

//NewEncoder allocates and initializes a new Encoder to compress sequences of Vec64
func NewEncoder(dst io.Writer, span int) *Encoder {
	enc := new(Encoder)
	enc.state = make([]elementState, span)
	enc.Reset(dst)
	return enc
}

//Reset the internal state of the encoder to write to a new given stream.
func (enc *Encoder) Reset(dst io.Writer) {
	for i := range enc.state {
		enc.state[i] = elementState{lead: 0xFF}
	}
	enc.writer = bitio.NewWriter(dst)
	enc.first = true
}

/*
Put encodes a Vec64 into a compressed, variable bits sequence
and writes the result to the underlying stream.

Put will panic if the given Vec64 has a different span than the encoder.
*/
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

//PutFloat64 is a short-hand for appending a vector of float64 to the encoder.
func (enc *Encoder) PutFloat64(vec []float64) error {
	sh := (*Vec64)(unsafe.Pointer(&vec))
	return enc.Put(*sh)
}

//PutUint64 is a short-hand for appending a vector of uint64 to the encoder.
func (enc *Encoder) PutUint64(vec []uint64) error {
	return enc.Put(Vec64(vec))
}

//Close appends an EOF bit sequence and
//flushes any remaining buffered data to the underlying stream
func (enc *Encoder) Close() error {
	const numControlBits = 2
	const eofBits = eofNumValueBits | (eofShiftBits << numValueBitsSize)
	enc.writer.TryWriteBits(eofBits, numControlBits+shiftSize+numValueBitsSize)
	if err := enc.writer.Close(); err != nil {
		return err
	}
	return enc.writer.TryError
}
