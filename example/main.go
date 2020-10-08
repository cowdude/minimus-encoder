package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand"

	minimus "github.com/cowdude/minimus-encoder"
)

func printFloats(s []float64) {
	for _, f := range s {
		fmt.Printf(" %7.4f", f)
	}
}

func getData() (res [][]float64) {
	res = make([][]float64, 1000, 1001)
	for i := range res {
		const pre = 100
		n := rand.Float64()
		n = math.Round(n*pre*pre) / pre
		res[i] = []float64{
			n,
			1 / (1 + n),
			float64(1 + (i/5)%42),
			float64(1 + i/13),
		}
	}
	res = append(res, []float64{1.11111, 2.22222, 3.33333, 4.44444})
	return
}

func printInputsOutputs(c chan minimus.Vec64, pool *minimus.VecPool, inputs [][]float64) {
	var i int
	const logLines = 2
	for vec := range c {
		if i < (logLines+1)/2 || i >= len(inputs)-logLines/2 {
			fmt.Printf("     %4d:", i)
			printFloats(inputs[i])
			fmt.Print("  =>")
			printFloats(vec.Float64())
			fmt.Println()
		}

		//recycle `vec` after using it
		pool.Put(vec)
		i++
	}
}

func lossyEncodeDecode(buf *bytes.Buffer, span int, maxError float64, series [][]float64) {
	var (
		enc = minimus.NewEncoder(buf, span)
		tmp = make([]float64, span)
	)
	for _, vec := range series {
		for i, val := range vec {
			tmp[i] = minimus.LossyFloat64(val, maxError)
		}
		enc.PutFloat64(tmp)
	}
	enc.Close()
	numBytes := buf.Len()

	var (
		pool = minimus.NewVecPool(span)
		c    = make(chan minimus.Vec64, 1)
		dec  = minimus.NewDecoder(buf, span)
		ctx  = context.TODO()
	)
	go func() {
		defer close(c)
		if err := dec.EnumBorrow(ctx, c, pool); err != nil {
			panic(err)
		}
	}()

	fmt.Println()
	printInputsOutputs(c, pool, series)
	bps := float64(numBytes*8) / float64(len(series)*span)
	fmt.Printf("|e|=%-5.e: %6.3f b/sample\n", maxError, bps)
}

func encodeDecode(buf *bytes.Buffer, span int, series [][]float64) {
	var (
		enc = minimus.NewEncoder(buf, span)
	)
	for _, vec := range series {
		enc.PutFloat64(vec)
	}
	enc.Close()
	numBytes := buf.Len()

	var (
		pool = minimus.NewVecPool(span)
		c    = make(chan minimus.Vec64, 1)
		dec  = minimus.NewDecoder(buf, span)
		ctx  = context.TODO()
	)
	go func() {
		defer close(c)
		if err := dec.EnumBorrow(ctx, c, pool); err != nil {
			panic(err)
		}
	}()

	fmt.Println()
	printInputsOutputs(c, pool, series)
	bps := float64(numBytes*8) / float64(len(series)*span)
	fmt.Printf("(lossless)    %.3f b/sample\n", bps)
}

func main() {
	var (
		buf    bytes.Buffer
		series = getData()
		span   = len(series[0])
	)
	for maxError := 1e-1; maxError > 1e-16; maxError *= 1e-3 {
		buf.Reset()
		lossyEncodeDecode(&buf, span, maxError, series)
	}

	buf.Reset()
	encodeDecode(&buf, span, series)
}
