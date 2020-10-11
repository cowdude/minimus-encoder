package minimus

import (
	"bytes"
	"io"
	"math"
	"math/rand"
	"testing"
)

const spanTest = 10

type dataset struct {
	name string
	ds   [][]float64
}

var datasets []dataset

func constData() dataset {
	n := rand.Float64()
	res := make([][]float64, 10000)
	for i := range res {
		res[i] = make([]float64, spanTest)
		for j := 0; j < spanTest; j++ {
			res[i][j] = n + float64(j)
		}
	}
	return dataset{"const", res}
}

func normRandData() dataset {
	res := make([][]float64, 10000)
	for i := range res {
		res[i] = make([]float64, spanTest)
		for j := 0; j < spanTest; j++ {
			res[i][j] = rand.Float64()
		}
	}
	return dataset{"norm rand", res}
}

func expRandData() dataset {
	res := make([][]float64, 10000)
	for i := range res {
		res[i] = make([]float64, spanTest)
		for j := 0; j < spanTest; j++ {
			res[i][j] = rand.ExpFloat64()
		}
	}
	return dataset{"exp rand", res}
}

func init() {
	const n = 5
	for i := 0; i < n; i++ {
		datasets = append(datasets, constData())
	}
	for i := 0; i < n; i++ {
		datasets = append(datasets, normRandData())
	}
	for i := 0; i < n; i++ {
		datasets = append(datasets, expRandData())
	}
}

func checkDecodedFloats(t *testing.T, ds, decoded [][]float64) {
	if len(ds) != len(decoded) {
		t.Logf("Length mismatch: %v instead of %v", len(decoded), len(ds))
		t.FailNow()
	}
	for i := range ds {
		for j := 0; j < spanTest; j++ {
			x, y := ds[i][j], decoded[i][j]
			if x != y && !math.IsNaN(x) && !math.IsNaN(y) {
				t.Logf("element mistmatch at %v;%v\n", i, j)
				t.Fail()
				break
			}
		}
	}
}

func testEncDecF64(t *testing.T, ds [][]float64) {
	span := len(ds[0])
	var buf bytes.Buffer
	enc := NewEncoder(&buf, span)
	for _, x := range ds {
		enc.PutFloat64(x)
	}
	enc.Close()
	dec := NewDecoder(&buf, span)
	var decoded [][]float64
	for dec.Next() {
		vec := dec.Current()
		cpy := make(Vec64, len(vec))
		copy(cpy, vec)
		decoded = append(decoded, cpy.Float64())
	}
	if dec.Err() != nil {
		panic(dec.Err())
	}
	checkDecodedFloats(t, ds, decoded)
}

func testCompressionRatio(t *testing.T, ds [][]float64, maxBPS float64, maxError float64) float64 {
	span := len(ds[0])
	var buf bytes.Buffer
	enc := NewEncoder(&buf, span)
	var hint BitHint
	for _, x := range ds {
		lossy := make([]float64, len(x))
		for i, n := range x {
			lossy[i], hint = LossyFloat64(n, maxError, hint)
		}
		enc.PutFloat64(lossy)
	}
	enc.Close()
	bitsPerSample := float64(buf.Len()*8) / float64(len(ds)*span)
	if bitsPerSample >= maxBPS {
		t.Fail()
		return math.NaN()
	}
	return bitsPerSample
}

func TestEncodeDecode(t *testing.T) {
	for _, ds := range datasets {
		t.Log(ds.name + "...")
		testEncDecF64(t, ds.ds)
	}
}

func TestCompressedBeatsPlainSize(t *testing.T) {
	for maxError := 1.0; maxError > 1e-15; maxError *= 1e-2 {
		t.Logf("Max error: %.2g", maxError)
		for i, ds := range datasets {
			name := ds.name
			if i > 0 && name == datasets[i-1].name {
				name = ""
			}
			t.Logf("   %-10v   %.3f b/sample", name,
				testCompressionRatio(t, ds.ds, 64.0, maxError))
		}
	}
}

func TestConstDataCompressionRatio(t *testing.T) {
	ds := constData()
	testCompressionRatio(t, ds.ds, 1.1, 0)
}

func TestConcatOutput(t *testing.T) {
	ds := expRandData().ds
	parts := [...][][]float64{
		ds[:len(ds)/2],
		ds[len(ds)/2:],
	}

	var buf bytes.Buffer
	span := len(ds[0])

	for _, part := range parts {
		enc := NewEncoder(&buf, span)
		for _, x := range part {
			enc.PutFloat64(x)
		}
		enc.Close()
	}

	dec := NewDecoder(&buf, span)
	var decoded [][]float64
	for dec.Next() {
		vec := dec.Current().Float64()
		cpy := make([]float64, len(vec))
		copy(cpy, vec)
		decoded = append(decoded, cpy)
	}
	if err := dec.Err(); err != nil {
		panic(err)
	}
	checkDecodedFloats(t, ds, decoded)
}

func TestDecodeEmptyStream(t *testing.T) {
	var buf bytes.Buffer
	dec := NewDecoder(&buf, 42)
	if dec.Next() {
		t.FailNow()
	}
	if err := dec.Err(); err != nil {
		t.Logf("Err not nil: %v", err)
		t.FailNow()
	}
}

func TestDecodePartialFirstVec(t *testing.T) {
	var buf bytes.Buffer
	//append first value (64bits)
	buf.Write([]byte{0x12, 0x34, 0x45, 0x78, 0x12, 0x34, 0x45, 0x78})
	//span is 2, but buffer only has a single element
	dec := NewDecoder(&buf, 2)
	if dec.Next() {
		t.FailNow()
	}
	if err := dec.Err(); err != io.EOF {
		t.Logf("Err not EOF: %v", err)
		t.FailNow()
	}
}

func TestDecodePartialNextVec(t *testing.T) {
	var buf bytes.Buffer
	//append first values (64bits)
	const span = 10
	for i := 0; i < span; i++ {
		buf.Write([]byte{0x12, 0x34, 0x45, 0x78, 0x12, 0x34, 0x45, 0x78})
	}
	//next vector (repeat last, missing 2 elements)
	buf.Write([]byte{0xFF})
	dec := NewDecoder(&buf, span)
	if !dec.Next() {
		t.FailNow()
	}
	if dec.Next() {
		t.FailNow()
	}
	if err := dec.Err(); err != io.EOF {
		t.Logf("Err not EOF: %v", err)
		t.FailNow()
	}
}
