package minimus

import (
	"sync"
	"unsafe"
)

type Vec64 []uint64

type VecPool struct{ p sync.Pool }

func (v Vec64) Float64() []float64 {
	return *(*[]float64)(unsafe.Pointer(&v))
}

func (v Vec64) Uint64() []uint64 {
	return []uint64(v)
}

func NewVecPool(span int) *VecPool {
	return &VecPool{sync.Pool{
		New: func() interface{} {
			return make(Vec64, span)
		},
	}}
}

func (p *VecPool) Get() Vec64  { return p.p.Get().(Vec64) }
func (p *VecPool) Put(v Vec64) { p.p.Put(v) }
