module fuzz

go 1.13

require (
	github.com/Illyrix/tidb-go-fuzz/dep v0.0.0-20201101090347-c89734463008
	github.com/Illyrix/tidb-go-fuzz/fuzz v0.0.0-20201104131545-cc5b91eb6de9
)

replace github.com/Illyrix/tidb-go-fuzz/dep => ../dep
