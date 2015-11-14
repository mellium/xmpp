.SILENT:

.PHONY: test
test:
	go test

.PHONY: benchmark
benchmark:
	go test -bench . -benchmem -run 'Benchmark.*'

.PHONY: build
build:
	go build
