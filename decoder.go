package minimus

import (
	"context"
	"errors"
	"io"

	"github.com/icza/bitio"
)

type decElementState struct {
	prevBits             uint64
	lshift, numValueBits uint8
}

type Decoder struct {
	reader  *bitio.Reader
	state   []decElementState
	outBits Vec64
	first   bool
}

func NewDecoder(src io.Reader, span int) *Decoder {
	return &Decoder{
		state:   make([]decElementState, span),
		outBits: make(Vec64, span),
		reader:  bitio.NewReader(src),
		first:   true,
	}
}

func (dec *Decoder) Next() bool {
	if dec.first {
		for i := range dec.outBits {
			bits := dec.reader.TryReadBits(floatBitsSize)
			dec.outBits[i] = bits
			dec.state[i].prevBits = bits
		}
		dec.first = false
		return true
	}

	for i := range dec.outBits {
		if dec.reader.TryReadBool() { //diff == 0 => repeat last
			dec.outBits[i] = dec.state[i].prevBits
			continue
		}
		if !dec.reader.TryReadBool() { //leading or trailing zeroes changed
			window := dec.reader.TryReadBits(shiftSize + numValueBitsSize)
			dec.state[i].lshift = uint8(window >> numValueBitsSize)
			dec.state[i].numValueBits = uint8(window & eofNumValueBits)
			if dec.state[i].lshift == eofShiftBits && dec.state[i].numValueBits == eofNumValueBits {
				if i != 0 {
					dec.reader.TryError = errors.New("partial vector at end of stream")
				}
				return false
			}
		}
		msb := dec.reader.TryReadBits(dec.state[i].numValueBits + 1)
		diff := msb << dec.state[i].lshift
		bits := dec.state[i].prevBits ^ diff
		dec.state[i].prevBits = bits
		dec.outBits[i] = bits
	}
	return true
}

func (dec *Decoder) EnumBorrow(ctx context.Context, out chan Vec64, pool *VecPool) error {
	for dec.Next() {
		vec := dec.Current()
		cpy := pool.Get()
		if len(vec) != len(cpy) {
			panic("incompatible vector shape in floats pool")
		}
		copy(cpy, vec)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- cpy:
		}
	}
	return dec.Err()
}

func (dec *Decoder) Err() error {
	return dec.reader.TryError
}

func (dec *Decoder) Current() Vec64 {
	return dec.outBits
}
