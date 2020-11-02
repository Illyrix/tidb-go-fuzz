// testing db config load

package filefuzz

import (
	"github.com/Illyrix/tidb-go-fuzz/fuzz/pkg/types"
)

type ConfFuzzer struct {
	types.Fuzzer
}
