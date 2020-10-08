package minimus

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

var benchData = expRandData().ds
var benchBuf bytes.Buffer
var encodedBuf []byte
var benchFloats []float64

func init() {
	benchBuf.Grow(1024 * 1024 * 64)

	enc := NewEncoder(&benchBuf, spanTest)
	for _, x := range benchData {
		enc.PutFloat64(x)
	}
	enc.Close()
	encodedBuf = make([]byte, benchBuf.Len())
	copy(encodedBuf, benchBuf.Bytes())
	fmt.Printf("Bench encoded data: %v bytes\n", len(encodedBuf))
}

func BenchmarkEncoding(b *testing.B) {
	benchBuf.Reset()
	enc := NewEncoder(&benchBuf, spanTest)
	defer enc.Close()
	for i := 0; i < b.N; i++ {
		enc.PutFloat64(benchData[i%len(benchData)])
	}
}

func BenchmarkEncodingLossy(b *testing.B) {
	benchBuf.Reset()
	enc := NewEncoder(&benchBuf, spanTest)
	defer enc.Close()
	var vec [spanTest]float64
	for i := 0; i < b.N; i++ {
		f := benchData[i%len(benchData)]
		for j := range vec {
			vec[j] = LossyFloat64(f[j], 1e-10)
		}
		enc.PutFloat64(vec[:])
	}
}

func BenchmarkDecoding(b *testing.B) {
	buf := bytes.NewReader(encodedBuf)
	var dec *Decoder
	for i := 0; i < b.N; i++ {
		if i == 0 || !dec.Next() {
			buf.Seek(0, io.SeekStart)
			dec = NewDecoder(buf, spanTest)
		}
		benchFloats = dec.Current().Float64()
	}
	if dec.Err() != nil {
		b.Fail()
	}
}
