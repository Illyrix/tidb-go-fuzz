package dep

import (
	"fmt"
	"net"
	"sync"

	"github.com/Illyrix/tidb-go-fuzz/dep/types"
)

var (
	traceTable *types.TraceBits
	mu         sync.Mutex // this lock is just for singleton
)

const ListenAddress = "127.0.0.1:16801"

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

// start listening in init()
func Listen() {
	handler := func(c *net.TCPConn) {
		data := make([]byte, 255)
		_, err := c.Read(data) // todo: support distinguish SQL trace log
		if err != nil {
			panic(err) // todo: debug
		}

		tb := GetTraceTable()
		tb.ClassifyCount()
		_, err = c.Write(tb.GetBits())
		if err != nil {
			panic(err) // todo: debug
		}

		tb.Clean()
	}

	go func() {
		socket, err := net.Listen("tcp", ListenAddress)
		if err != nil {
			panic(fmt.Sprintf("start linstening failed: %v", err))
		}
		defer socket.Close()
		for {
			conn, err := socket.Accept()
			if err != nil {
				panic(err) // todo: debug
			}

			// Note: it can be common function call
			// because only one connection at one time
			go handler(conn.(*net.TCPConn))
		}
	}()
}
