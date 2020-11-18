package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Illyrix/tidb-go-fuzz/fuzz/pkg"
	"github.com/Illyrix/tidb-go-fuzz/fuzz/pkg/builder"
	"github.com/Illyrix/tidb-go-fuzz/fuzz/pkg/types"
)

var flagIsRemote = flag.Bool("remote", false, "using remote tidb repo")
var flagSrcDir = flag.String("src", "", "path to local tidb repo")
var flagTargetDir = flag.String("target", "/tmp/tidb-go-fuzz", "path to modified tidb source code; should be empty")

var ignoreFiles map[string]struct{} = make(map[string]struct{})
var void struct{}

func main() {
	flag.Parse()

	config := types.Config{
		TidbSrcDir:     *flagSrcDir,
		TidbFromRemote: *flagIsRemote,
		TidbTargetDir:  *flagTargetDir,
	}

	if err := config.Valid(); err != nil {
		panic(err)
	}

	err := os.MkdirAll(*flagTargetDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Fatal Error: target dir %s create fail %v\n", *flagTargetDir, err)
	}

	// init filter map
	ignoreFiles[filepath.Join(*flagTargetDir, ".idea")] = void
	ignoreFiles[filepath.Join(*flagTargetDir, ".git")] = void
	ignoreFiles[filepath.Join(*flagTargetDir, ".vscode")] = void

	// copy tidb source code to target dir
	pkg.Copy(*flagSrcDir, *flagTargetDir)

	// walk on every file
	err = filepath.Walk(*flagTargetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if _, ok := ignoreFiles[path]; ok {
			return filepath.SkipDir
		}

		if !info.IsDir() &&
			strings.HasSuffix(info.Name(), ".go") &&
			!strings.HasSuffix(info.Name(), "_test.go") {
			src, err := ioutil.ReadFile(path)
			if err != nil {
				panic(path + " read error\n")
			}
			modifiedFile := addCounter(src)
			err = ioutil.WriteFile(path, modifiedFile, os.ModePerm)
			if err != nil {
				panic(fmt.Sprintf("%s write error %v\n", path, err))
			}
		}
		return nil
	})
	if err != nil {
		panic("walk files for adding counters failed")
	}

	// add listen in tidb-server/main.go
	builder.AddListenStart(*flagTargetDir)

	// install dependency
	fmt.Println("Installing dependency")
	builder.InstallDep(*flagTargetDir)

	fmt.Println("Compiling tidb")
	builder.CompileTidb(*flagTargetDir)

	fmt.Printf("Done! Run `%s` to start tidb server", "")
}

func addCounter(src []byte) []byte {
	fset, astFile := parse(src)

	visitor := builder.NewVisitorPtr(fset)
	ast.Walk(visitor, astFile)
	if visitor.Changed {
		visitor.AddImportDecl(astFile)
	}

	out := new(bytes.Buffer)
	cfg := printer.Config{
		Mode:     printer.SourcePos,
		Tabwidth: 8,
		Indent:   0,
	}
	cfg.Fprint(out, fset, astFile)
	return out.Bytes()
}

func parse(content []byte) (*token.FileSet, *ast.File) {
	fset := token.NewFileSet()
	aFile, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	return fset, aFile
}
