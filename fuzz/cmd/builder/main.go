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
	"regexp"
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
			// keep build constaints like: // +build linux
			buildComments := findCrucialComments(src)
			modifiedFile := addCounter(src)
			if len(buildComments) > 0 {
				// add build constaints back to source file
				modifiedFile = addBackComments(buildComments, modifiedFile)
			}
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
		Mode:     0,
		Tabwidth: 8,
		Indent:   0,
	}
	cfg.Fprint(out, fset, astFile)
	return out.Bytes()
}

func parse(content []byte) (*token.FileSet, *ast.File) {
	fset := token.NewFileSet()
	aFile, err := parser.ParseFile(fset, "", content, 0)
	if err != nil {
		panic(err)
	}

	return fset, aFile
}

// get all build constraints
func findCrucialComments(src []byte) []buildComment {
	str := string(src[:])
	lines := strings.Split(str, "\n")
	res := make([]buildComment, 0)

	buildReg := regexp.MustCompile(`// \+build .+`)
	generateReg := regexp.MustCompile(`//go:generate .+`)
	linknameReg := regexp.MustCompile(`//go:linkname .+`)

	for idx, line := range lines {
		if buildReg.MatchString(line) {
			res = append(res, buildComment{true, "", "", line + "\n\n"})
			continue
		}
		if generateReg.MatchString(line) {
			res = append(res, buildComment{true, "", "", line + "\n\n"})
			continue
		}
		if linknameReg.MatchString(line) {
			if len(lines) <= idx+1 {
				panic(fmt.Sprintf("no line after //go:linkname at line %d", idx))
			}
			nextLine := lines[idx+1]
			res = append(res, buildComment{false, nextLine, "", "\n" + line})
		}
	}
	return res
}

func addBackComments(comments []buildComment, src []byte) []byte {
	res := string(src)
	for _, comment := range comments {
		if comment.atHead {
			res = comment.comment + res
			continue
		}
		if comment.before != "" {
			lines := strings.Split(res, "\n")
			for idx, line := range lines {
				if line == comment.before {
					// before := lines[:idx-1]
					before := make([]string, idx)
					copy(before, lines[:idx])
					fmt.Println("  BEFORE: " + before[len(before)-1])
					before = append(before, comment.comment)
					fmt.Println("  INSERT: " + comment.comment)
					fmt.Println("  COMMENT_BEFORE:" + comment.before)
					after := lines[idx:]
					fmt.Println("  AFTER: " + after[0])
					res = strings.Join(append(before, after...), "\n")
					break
				}
				continue
			}
		} else if comment.after != "" {
			lines := strings.Split(res, "\n")
			for idx, line := range lines {
				if line == comment.after {
					before := lines[:idx-1]
					before = append(before, comment.comment)
					after := lines[idx-1:]
					res = strings.Join(append(before, after...), "\n")
					break
				}
				continue
			}
		}
	}
	return []byte(res)
}

type buildComment struct {
	atHead  bool
	before  string // this comment is before some line
	after   string // this comment is after some line
	comment string
}
