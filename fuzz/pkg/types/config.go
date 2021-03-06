package types

import (
	"errors"

	"github.com/Illyrix/tidb-go-fuzz/fuzz/pkg"
)

const TIDB_REMOTE_URL = "https://github.com/pingcap/tidb"

type Config struct {
	TidbSrcDir     string // local tidb source code; ignored if `TidbFromRemote` is true
	TidbFromRemote bool   // using current master branch from github/tidb
	TidbTargetDir  string // where we copy tidb source code to; should be empty

	// todo: other fuzzer configures
}

func (c *Config) Valid() error {
	if c.TidbSrcDir == "" && !c.TidbFromRemote {
		return errors.New("directory of source code is not assigned")
	}
	if pkg.DirExists(c.TidbTargetDir) {
		return errors.New("target tidb code dir exists")
	}
	return nil
}
