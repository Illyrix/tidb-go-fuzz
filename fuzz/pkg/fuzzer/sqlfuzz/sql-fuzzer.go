// test

package sqlfuzz

import (
	"github.com/Illyrix/tidb-go-fuzz/fuzz/pkg/types"
)

type SQLFuzzer struct {
	types.Fuzzer

	sql  string
	conn db.Conn // db connection
}
