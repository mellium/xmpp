PACKAGES=$$(go list ./...)

.PHONEY: test
test: vendor
	go test -cover $(PACKAGES) -race

.PHONEY: bench
bench: vendor
	go test -bench . -benchmem -run NONE $(PACKAGES)

.PHONEY: vet
vet:
	go vet $(PACKAGES)

vendor: Gopkg.toml
	dep ensure

deps.svg: *.go
	(   echo "digraph G {"; \
	go list -f '{{range .Imports}}{{printf "\t%q -> %q;\n" $$.ImportPath .}}{{end}}' \
		$$(go list -f '{{join .Deps " "}}' .) .; \
	echo "}"; \
	) | dot -Tsvg -o $@
