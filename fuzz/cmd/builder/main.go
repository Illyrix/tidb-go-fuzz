package main

import (
	"flag"

	"github.com/Illyrix/tidb-go-fuzz/fuzz/pkg/types"
)

var flagIsRemote = flag.Bool("remote", false, "using remote tidb repo")
var flagSrcDir = flag.String("src", "", "path to local tidb repo")
var flagTargetDir = flag.String("target", "/tmp/tidb-go-fuzz", "path to modified tidb source code; will be created or cleaned")

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
}
