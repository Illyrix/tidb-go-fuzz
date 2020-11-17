module github.com/Illyrix/tidb-go-fuzz/fuzz

go 1.13

require (
	github.com/Illyrix/tidb-go-fuzz/dep v0.0.0-20201117064616-9ae239322e6a
	github.com/stretchr/testify v1.6.1
)

replace github.com/Illyrix/tidb-go-fuzz/dep => ../dep
