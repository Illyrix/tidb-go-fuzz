package types

import "errors"

type BlockIdType = uint16
type TraceRouteType = uint16         // same as block id
const TraceBitsSize uint64 = 1 << 16 // is 1 << 32 if BlockIdType is uint32

type TraceBits [TraceBitsSize]byte

func NewTraceBits() *TraceBits {
	var tb TraceBits
	return &tb
}

// src will be lsift 1 in building stage; avoid cases like A^A=0, A^B=B^A
func (tb *TraceBits) GetCount(src, dst BlockIdType) (uint8, error) {
	if tb == nil {
		return 0, errors.New("TraceBits has not been initialized")
	}
	return (*tb)[src^dst], nil
}

// see: http://rk700.github.io/2017/12/28/afl-internals/#%E5%88%86%E6%94%AF%E4%BF%A1%E6%81%AF%E7%9A%84%E5%88%86%E6%9E%90
func (tb *TraceBits) ClassifyCount() error {
	if tb == nil {
		return errors.New("TraceBits has not been initialized")
	}
	for key, val := range *tb {
		switch {
		case val < 3:
			break
		case val == 3:
			(*tb)[key] = 4
		case val < 8:
			(*tb)[key] = 8
		case val < 16:
			(*tb)[key] = 16
		case val < 32:
			(*tb)[key] = 32
		case val < 128:
			(*tb)[key] = 64
		default:
			(*tb)[key] = 128
		}
	}
	return nil
}

func (tb *TraceBits) AddCount(src, dst BlockIdType) {
	key := src ^ dst
	if (*tb)[key] != 255 { // avoid overflow
		(*tb)[key]++
	}
}
