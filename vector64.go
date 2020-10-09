package minimus

import (
	"sync"
	"unsafe"
)

//Vec64 represents a N vector of 64-bit primitives
type Vec64 []uint64

//VecPool is a concurrent pool of Vec64
type VecPool struct{ p sync.Pool }

//Float64 casts a Vec64 as a float64 slice without copying it
func (v Vec64) Float64() []float64 {
	return *(*[]float64)(unsafe.Pointer(&v))
}

//Uint64 casts a Vec64 as a uint64 slice without copying it
func (v Vec64) Uint64() []uint64 {
	return []uint64(v)
}

//NewVecPool creates a new empty VecPool
func NewVecPool(span int) *VecPool {
	return &VecPool{sync.Pool{
		New: func() interface{} {
			return make(Vec64, span)
		},
	}}
}

//Get takes or allocates a new Vec64 from the pool
func (p *VecPool) Get() Vec64 { return p.p.Get().(Vec64) }

//Put returns an existing Vec64 to the pool
func (p *VecPool) Put(v Vec64) { p.p.Put(v) }
