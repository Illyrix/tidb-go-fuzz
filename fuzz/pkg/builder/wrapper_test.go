package builder

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Illyrix/tidb-go-fuzz/dep/types"
	"github.com/stretchr/testify/assert"
)

func AstToBytes(node ast.Node, fset *token.FileSet) *bytes.Buffer {
	out := new(bytes.Buffer)
	cfg := printer.Config{
		Mode:     printer.SourcePos,
		Tabwidth: 8,
		Indent:   0,
	}
	cfg.Fprint(out, fset, node)
	return out
}

const code = `
	package test
	import "fmt"
	func main() {
	}
`

func TestMakeCountNode(t *testing.T) {
	var src types.BlockIdType = 0x0001
	var dst types.BlockIdType = 0x31AF

	stmt := makeCountNode(src, dst)

	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", code, parser.ParseComments)

	assert.Equal(t, nil, err)

	out := AstToBytes(stmt, fset)

	assert.Equal(t, out.String(), "__tidb_go_fuzz_dep.GetTraceTable().AddCount(1, 12719)")
}

func TestAddListenStart(t *testing.T) {
	tempTidbSrc := "/tmp/fuzz-tidb"
	tidbServerGoFile := `
	package main

	import (
		"context"
		"flag"
		"fmt"
		"io/ioutil"
		"os"
		"runtime"
		//...
	)

	const (
		nmVersion                = "V"
		nmConfig                 = "config"
		nmConfigCheck            = "config-check"
		nmConfigStrict           = "config-strict"
		nmStore                  = "store"
		//...
	)

	var (
		version      = flagBoolean(nmVersion, false, "print version information and exit")
		configPath   = flag.String(nmConfig, "", "config file path")
		configCheck  = flagBoolean(nmConfigCheck, false, "check config file validity and exit")
		configStrict = flagBoolean(nmConfigStrict, false, "enforce config file validity")
		//...
	)

	var (
		storage  kv.Storage
		dom      *domain.Domain
		svr      *server.Server
		graceful bool
	)

	func main() {
		flag.Parse()
		//...
		setGlobalVars()
		setCPUAffinity()
		setupLog()
		setHeapProfileTracker()
		setupTracing() // Should before createServer and after setup config.
		//...
	}
	func exit() {
		syncLog()
		os.Exit(0)
	}
	
	func syncLog() {
		if err := log.Sync(); err != nil {
			fmt.Fprintln(os.Stderr, "sync log err:", err)
			os.Exit(1)
		}
	}
	`
	defer func() {
		if err := os.RemoveAll(tempTidbSrc); err != nil {
			panic(err)
		}
	}()

	realPath := filepath.Join(tempTidbSrc, "tidb-server")
	if err := os.MkdirAll(realPath, 0777); err != nil {
		panic(err)
	}
	realPath = filepath.Join(realPath, "main.go")
	if err := ioutil.WriteFile(realPath, []byte(tidbServerGoFile), 0777); err != nil {
		panic(err)
	}
	AddListenStart(tempTidbSrc)

	content, err := ioutil.ReadFile(realPath)
	if err != nil {
		panic(err)
	}
	lines := strings.Fields(string(content[:]))
	checkStr := lines[87] + lines[91]
	assert.Equal(t, checkStr, "__tidb_go_fuzz_dep.Listen()")
}
