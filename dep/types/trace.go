package types

import (
	"errors"
	"sync"
)

type BlockIdType = uint16
type TraceRouteType = uint16         // same as block id
const TraceBitsSize uint64 = 1 << 16 // is 1 << 32 if BlockIdType is uint32

type TraceBits struct {
	bits [TraceBitsSize]byte
	mu   sync.RWMutex
}

func NewTraceBits() *TraceBits {
	return &TraceBits{}
}

func (tb *TraceBits) GetCount(src, dst BlockIdType) (uint8, error) {
	if tb == nil {
		return 0, errors.New("TraceBits has not been initialized")
	}
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.bits[src^dst], nil
}

func (tb *TraceBits) GetBits() [TraceBitsSize]byte {
	if tb == nil {
		return 0, errors.New("TraceBits has not been initialized")
	}
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.bits
}

// see: http://rk700.github.io/2017/12/28/afl-internals/#%E5%88%86%E6%94%AF%E4%BF%A1%E6%81%AF%E7%9A%84%E5%88%86%E6%9E%90
func (tb *TraceBits) ClassifyCount() error {
	if tb == nil {
		return errors.New("TraceBits has not been initialized")
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
	for key, val := range tb.bits {
		switch {
		case val < 3:
			break
		case val == 3:
			tb.bits[key] = 4
		case val < 8:
			tb.bits[key] = 8
		case val < 16:
			tb.bits[key] = 16
		case val < 32:
			tb.bits[key] = 32
		case val < 128:
			tb.bits[key] = 64
		default:
			tb.bits[key] = 128
		}
	}
	return nil
}

// src will be lsift 1 in building stage; avoid cases like A^A=0, A^B=B^A
func (tb *TraceBits) AddCount(src, dst BlockIdType) {
	if tb == nil {
		panic("TraceBits has not been initialized")
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()

	key := (src << 1) ^ dst
	if tb.bits[key] != 255 { // avoid overflow
		tb.bits[key]++
	}
}

func (tb *TraceBits) Clean() {
	if tb == nil {
		panic("TraceBits has not been initialized")
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.bits = [TraceBitsSize]byte{0}
}
