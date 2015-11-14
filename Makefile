.SILENT:

.PHONY: test
test:
	go test -cover

.PHONY: benchmark
benchmark:
	go test -cover -bench . -benchmem -run 'Benchmark.*'

.PHONY: build
build:
	go build
