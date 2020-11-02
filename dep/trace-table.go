package dep

import (
	"sync"

	"github.com/Illyrix/tidb-go-fuzz/dep/types"
)

var traceTable *types.TraceBits
var mu sync.Mutex

// singleton
// todo: a map of TraceBits instead of only one
// NOTE: it only support one SQL execution at one time currently
// because it's hard to distinguish which SQL

func GetTraceTable() *types.TraceBits {
	mu.Lock()
	defer mu.Unlock()

	if traceTable == nil {
		traceTable = types.NewTraceBits()
	}

	return traceTable
}
