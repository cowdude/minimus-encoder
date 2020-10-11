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

//Decoder reads a compressed data stream and allows iterating over the
//resulting decoded sequence of Vec64
type Decoder struct {
	reader  *bitio.Reader
	state   []decElementState
	outBits Vec64
	first   bool
}

//NewDecoder allocates and initializes a new Decoder
func NewDecoder(src io.Reader, span int) *Decoder {
	return &Decoder{
		state:   make([]decElementState, span),
		outBits: make(Vec64, span),
		reader:  bitio.NewReader(src),
		first:   true,
	}
}

//Next decodes the next Vec64 element from the stream.
//Returns false if an error happens, or end of stream is reached.
func (dec *Decoder) Next() bool {
nextVector:
	for {
		if dec.first {
			dec.first = false
			for i := range dec.outBits {
				bits := dec.reader.TryReadBits(floatBitsSize)
				if i == 0 && dec.reader.TryError == io.EOF {
					dec.reader.TryError = nil
					return false
				}
				dec.outBits[i] = bits
				dec.state[i].prevBits = bits
			}
			break
		} else {
			for i := range dec.outBits {
				if dec.reader.TryReadBool() { //diff == 0 => repeat last
					dec.outBits[i] = dec.state[i].prevBits
					continue
				}
				if i == 0 && dec.reader.TryError == io.EOF {
					dec.reader.TryError = nil
					return false
				}
				if !dec.reader.TryReadBool() { //leading or trailing zeroes changed
					window := dec.reader.TryReadBits(shiftSize + numValueBitsSize)
					dec.state[i].lshift = uint8(window >> numValueBitsSize)
					dec.state[i].numValueBits = uint8(window & eofNumValueBits)
					if dec.state[i].lshift == eofShiftBits && dec.state[i].numValueBits == eofNumValueBits {
						if i != 0 {
							dec.reader.TryError = errors.New("partial vector")
							return false
						}
						//reset state and skip current element
						dec.reader.Align()
						dec.first = true
						continue nextVector
					}
				}
				msb := dec.reader.TryReadBits(dec.state[i].numValueBits + 1)
				diff := msb << dec.state[i].lshift
				bits := dec.state[i].prevBits ^ diff
				dec.state[i].prevBits = bits
				dec.outBits[i] = bits
			}
			break
		}
	}
	return dec.reader.TryError == nil
}

/*
EnumBorrow is a helper method for enumerating Vec64s into a go channel.

Channel elements should be returned to the pool when no longer needed.
*/
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

//Err returns the last error after calling Next.
//Err returns nil at EOF, not io.EOF.
func (dec *Decoder) Err() error {
	return dec.reader.TryError
}

//Current returns the last Vec64 decoded after calling Next.
//Only valid if Next returned true.
func (dec *Decoder) Current() Vec64 {
	return dec.outBits
}
