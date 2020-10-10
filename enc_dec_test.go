package minimus

import (
	"bytes"
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
	if len(ds) != len(decoded) {
		t.Log("Length mismatch")
		t.Fail()
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
