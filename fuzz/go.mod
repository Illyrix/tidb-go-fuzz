module github.com/Illyrix/tidb-go-fuzz/fuzz

go 1.13

require (
	github.com/Illyrix/tidb-go-fuzz/dep v0.0.0-20201118183325-aa3f46ee5bea
	github.com/stretchr/testify v1.6.1
)

replace github.com/Illyrix/tidb-go-fuzz/dep => ../dep
