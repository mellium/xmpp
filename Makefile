PACKAGES=$$(go list ./... | grep -v '/vendor/')

.PHONEY: test
test:
	go test -cover $(PACKAGES) -race

.PHONEY: bench
bench:
	go test -bench . -benchmem -run NONE $(PACKAGES)

.PHONEY: vet
vet:
	go vet $(PACKAGES)

deps.svg: *.go
	(   echo "digraph G {"; \
	go list -f '{{range .Imports}}{{printf "\t%q -> %q;\n" $$.ImportPath .}}{{end}}' \
		$$(go list -f '{{join .Deps " "}}' .) .; \
	echo "}"; \
	) | dot -Tsvg -o $@
