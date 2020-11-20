package builder

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/Illyrix/tidb-go-fuzz/dep/types"
)

// `import github.com/Illyrix/tidb-go-fuzz-dep as "tidb_go_fuzz_dep"``
const FUZZ_DEP_IMPORT_AS = "__tidb_go_fuzz_dep"
const FUZZ_DEP_IMPORT_NAME = "github.com/Illyrix/tidb-go-fuzz/dep"

// then use `__tidb_go_fuzz_dep.GetTraceTable()` to fetch the singleton
// TraceTable instance

func makeCountNode(src, dst types.BlockIdType) ast.Stmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.Ident{
							Name: FUZZ_DEP_IMPORT_AS,
						},
						Sel: &ast.Ident{
							Name: "GetTraceTable",
						},
					},
				},
				Sel: &ast.Ident{
					Name: "AddCount",
				},
			},
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.INT,
					Value: strconv.FormatUint(uint64(src), 10),
				},
				&ast.BasicLit{
					Kind:  token.INT,
					Value: strconv.FormatUint(uint64(dst), 10),
				},
			},
		},
	}
}

// `go add .../tidb-go-fuzz/dep`
func InstallDep(root string) {
	shellCmd := exec.Command("go", "get", "-u", "github.com/Illyrix/tidb-go-fuzz/dep")
	shellCmd.Dir = root
	buf := &bytes.Buffer{}
	shellCmd.Stdout = buf
	err := shellCmd.Run()
	if err != nil {
		panic(fmt.Sprintf("go get error %v\n%s\n", err, buf.String()))
	}
}

func CompileTidb(root string) {
	shellCmd := exec.Command("make", "server")
	shellCmd.Dir = root
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	shellCmd.Stdout = buf
	shellCmd.Stderr = errBuf
	err := shellCmd.Run()
	if err != nil {
		panic(fmt.Sprintf("compile error %v\n%s\n%s", err, buf.String(), errBuf.String()))
	}
}

// inject calling `tidb_go_fuzz.Listen()` on startup
func AddListenStart(root string) {
	// located at tidb-server/main.go
	main := filepath.Join(root, "tidb-server", "main.go")
	fset := token.NewFileSet()

	content, err := ioutil.ReadFile(main)
	if err != nil {
		panic(main + " read error\n")
	}
	aFile, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	for _, decl := range aFile.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Name.Name != "main" {
			continue
		}

		callListen := &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: FUZZ_DEP_IMPORT_AS,
					},
					Sel: &ast.Ident{
						Name: "Listen",
					},
				},
			},
		}

		funcDecl.Body.List = append([]ast.Stmt{callListen}, funcDecl.Body.List...)
	}

	out := new(bytes.Buffer)
	cfg := printer.Config{
		Mode:     printer.SourcePos,
		Tabwidth: 8,
		Indent:   0,
	}
	cfg.Fprint(out, fset, aFile)

	// write back to file
	err = ioutil.WriteFile(main, out.Bytes(), os.ModePerm)
	if err != nil {
		log.Fatalf("%s write error %v\n", main, err)
	}
}
