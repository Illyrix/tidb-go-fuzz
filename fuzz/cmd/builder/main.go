package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

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

}
